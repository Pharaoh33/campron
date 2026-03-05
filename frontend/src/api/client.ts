import axios from 'axios'

export const apiBase = import.meta.env.VITE_API_BASE as string

export const http = axios.create({
  baseURL: apiBase,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' }
})

http.interceptors.response.use(
  r => r,
  err => {
    // keep error readable
    return Promise.reject(err)
  }
)
