<script setup lang="ts">
/**
 * 前端主页面（Vue3 + TS）
 * - 输入单词、选择口音
 * - 回车或点击按钮调用后端接口下载 mp3 + 音标
 * - 展示下载结果（IPA + 可访问 mp3 URL + 本地保存目录）
 */
import { ref } from 'vue'
import { apiBase } from './api/client'
import { downloadPronunciation } from './api/pronunciation'
import type { Accent, SavedItem } from './api/types'

const word = ref('')
const accent = ref<Accent>('us')
const loading = ref(false)
const errMsg = ref('')
const pageUrl = ref('')
const saved = ref<SavedItem[]>([])

/** 触发下载：支持点击按钮或在输入框回车触发 */
async function onSubmit() {
  errMsg.value = ''
  pageUrl.value = ''
  saved.value = []

  const w = word.value.trim()
  if (!w) {
    errMsg.value = '请输入单词'
    return
  }

  loading.value = true
  try {
    // 调用后端：POST /api/v1/pronunciations/download
    const res = await downloadPronunciation({ word: w, accent: accent.value })

    if (!res.ok) {
      errMsg.value = res.message || '下载失败'
      pageUrl.value = res.page_url || ''
      return
    }

    pageUrl.value = res.page_url || ''
    saved.value = res.saved || []
  } catch (e: any) {
    // 尽量把后端返回的 message 展示给用户
    errMsg.value = e?.response?.data?.message || e?.message || '请求失败'
  } finally {
    loading.value = false
  }
}

/** 输入框按下回车：触发下载（避免在 IME 组合输入时误触） */
function onWordKeydown(e: KeyboardEvent) {
  // isComposing：中文输入法正在选词时不触发
  // key === 'Enter'：按下回车
  // loading：请求中不重复触发
  // @ts-ignore
  if ((e as any).isComposing) return
  if (e.key === 'Enter' && !loading.value) {
    e.preventDefault()
    onSubmit()
  }
}
</script>

<template>
  <div class="container">
    <div class="header">
      <div class="title">CamPron</div>
      <div class="muted">API: <span class="badge">{{ apiBase }}</span></div>
    </div>

    <div class="card">
      <!-- 用 form 包起来：更符合真实业务页面习惯（回车提交） -->
      <form class="row" @submit.prevent="onSubmit">
        <input
          v-model="word"
          placeholder="例如：activity"
          @keydown="onWordKeydown"
        />
        <select v-model="accent">
          <option value="us">US</option>
          <option value="uk">UK</option>
          <option value="both">BOTH</option>
        </select>
        <button type="submit" :disabled="loading">
          {{ loading ? '下载中…' : '下载 mp3 + 音标' }}
        </button>
      </form>

      <div class="muted" style="margin-top:10px">
        会在 <code>storage.download_dir</code> 下自动创建以单词命名的文件夹（例如 <code>downloads/activity/</code>），里面包含 mp3 和音标 txt。<br />
        音标格式与 Cambridge 页面一致（例如 <code>/ækˈtɪv.ə.ti/</code>）。
      </div>

      <div v-if="errMsg" class="log" style="margin-top:14px">
        {{ errMsg }}
        <div v-if="pageUrl">page_url: {{ pageUrl }}</div>
      </div>

      <div v-if="saved.length" style="margin-top:14px">
        <div class="muted" v-if="pageUrl">
          来源页面：<a :href="pageUrl" target="_blank" rel="noreferrer">{{ pageUrl }}</a>
        </div>

        <table class="table">
          <thead>
            <tr>
              <th>Accent</th>
              <th>IPA（与 Cambridge 一致）</th>
              <th>MP3 URL</th>
              <th>Local Folder</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in saved" :key="s.mp3_filename">
              <td><span class="badge">{{ s.accent.toUpperCase() }}</span></td>
              <td><code>{{ s.ipa || '-' }}</code></td>
              <td><a :href="s.mp3_url" target="_blank" rel="noreferrer">{{ s.mp3_url }}</a></td>
              <td><code>{{ s.folder }}</code></td>
            </tr>
          </tbody>
        </table>

        <div class="muted" style="margin-top:10px">
          文件结构示例：<br />
          <code>.../downloads/activity/activity_us.mp3</code><br />
          <code>.../downloads/activity/activity_us.ipa.txt</code>
        </div>
      </div>
    </div>
  </div>
</template>
