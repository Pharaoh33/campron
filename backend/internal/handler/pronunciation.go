package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"campron_enterprise/backend/internal/config"
	"campron_enterprise/backend/internal/service/cambridge"
	"campron_enterprise/backend/internal/util"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
企业常见做法：
- Handler 只负责：参数校验、调用 Service、拼装响应
- 业务逻辑（抓网页/解析/下载）放到 Service 层
- 保存路径等可变参数从配置读取
*/

type DownloadReq struct {
	Word   string `json:"word" binding:"required"` // 必填：单词
	Accent string `json:"accent"`                  // 选填：us / uk / both（默认 us）
}

type SavedItem struct {
	Accent      string `json:"accent"`       // uk/us
	Folder      string `json:"folder"`       // 单词文件夹（绝对路径）
	MP3Filename string `json:"mp3_filename"` // mp3 文件名
	MP3Path     string `json:"mp3_path"`     // mp3 绝对路径
	MP3URL      string `json:"mp3_url"`      // 供前端访问的 URL（后端静态文件路由）
	IPA         string `json:"ipa"`          // 音标（尽量保持与 Cambridge 页面一致，例如 /ækˈtɪv.ə.ti/）
	IPAFilename string `json:"ipa_filename"` // ipa txt 文件名
	IPAPath     string `json:"ipa_path"`     // ipa txt 绝对路径
}

type DownloadResp struct {
	OK      bool        `json:"ok"`
	Message string      `json:"message,omitempty"`
	PageURL string      `json:"page_url,omitempty"` // Cambridge 词条页面
	Saved   []SavedItem `json:"saved,omitempty"`    // 成功保存的条目（可能部分成功）
}

func DownloadPronunciation(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	svc := cambridge.New(cfg)

	return func(c *gin.Context) {
		// 1) 参数校验
		var req DownloadReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, DownloadResp{OK: false, Message: "invalid request"})
			return
		}

		word := strings.TrimSpace(req.Word)
		accent := strings.ToLower(strings.TrimSpace(req.Accent))
		if accent == "" {
			accent = "us"
		}
		if accent != "us" && accent != "uk" && accent != "both" {
			c.JSON(http.StatusBadRequest, DownloadResp{OK: false, Message: "accent must be us/uk/both"})
			return
		}

		// 2) 抓取词条 HTML
		pageURL, html, err := svc.FetchEntryHTML(word)
		if err != nil {
			logger.Warn("fetch_failed", zap.Error(err))
			c.JSON(http.StatusBadGateway, DownloadResp{OK: false, Message: "fetch failed", PageURL: pageURL})
			return
		}

		// 3) 解析 mp3 URL
		mp3s, err := svc.ExtractMP3URLs(html, accent)
		if err != nil {
			logger.Warn("parse_mp3_failed", zap.Error(err))
			c.JSON(http.StatusBadGateway, DownloadResp{OK: false, Message: "parse mp3 failed", PageURL: pageURL})
			return
		}
		if len(mp3s) == 0 {
			c.JSON(http.StatusNotFound, DownloadResp{OK: false, Message: "no mp3 found", PageURL: pageURL})
			return
		}

		// 4) 解析 IPA（启发式：尽量按 uk/us 区域块找）
		ipas := svc.ExtractIPA(html, accent)

		// 5) 为单词创建独立文件夹：<download_dir>/<word>/
		folderName := util.SafeFilename(word)
		wordDir := filepath.Join(cfg.Storage.DownloadDir, folderName)
		if err := os.MkdirAll(wordDir, 0o755); err != nil {
			logger.Warn("mkdir_failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, DownloadResp{OK: false, Message: "failed to create word folder", PageURL: pageURL})
			return
		}

		// 6) 下载 mp3 + 写入 ipa txt（允许部分成功）
		saved := make([]SavedItem, 0, len(mp3s))
		for _, u := range mp3s {
			acc := u.Accent

			// mp3 命名：<word>_<accent>.mp3
			mp3Filename := folderName + "_" + acc + ".mp3"
			mp3Path := filepath.Join(wordDir, mp3Filename)

			if err := svc.DownloadMP3(u.URL, pageURL, mp3Path); err != nil {
				logger.Warn("download_mp3_failed", zap.String("accent", acc), zap.Error(err))
				continue
			}

			// ipa txt 命名：<word>_<accent>.ipa.txt
			ipa := strings.TrimSpace(ipas[acc])
			ipaFilename := ""
			ipaPath := ""
			if ipa != "" {
				ipaFilename = folderName + "_" + acc + ".ipa.txt"
				ipaPath = filepath.Join(wordDir, ipaFilename)
				// 写入原样音标（末尾补换行，方便命令行查看）
				if err := os.WriteFile(ipaPath, []byte(ipa+""), 0o644); err != nil {
					logger.Warn("write_ipa_failed", zap.String("accent", acc), zap.Error(err))
					ipaFilename = ""
					ipaPath = ""
				}
			}

			// 7) 静态文件访问 URL：/files 直接映射到 download_dir，所以需要包含 word 子目录
			base := strings.TrimRight(cfg.Server.BaseURL, "/")
			mp3URL := base + "/files/" + folderName + "/" + mp3Filename

			saved = append(saved, SavedItem{
				Accent:      acc,
				Folder:      wordDir,
				MP3Filename: mp3Filename,
				MP3Path:     mp3Path,
				MP3URL:      mp3URL,
				IPA:         ipa,
				IPAFilename: ipaFilename,
				IPAPath:     ipaPath,
			})
		}

		if len(saved) == 0 {
			c.JSON(http.StatusBadGateway, DownloadResp{OK: false, Message: "download failed (maybe blocked)", PageURL: pageURL})
			return
		}

		c.JSON(http.StatusOK, DownloadResp{OK: true, PageURL: pageURL, Saved: saved})
	}
}
