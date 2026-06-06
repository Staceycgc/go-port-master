import axios from 'axios'
import { ElMessage } from 'element-plus'
import { clearAuthToken, getAuthToken } from '@/utils/auth'

const request = axios.create({
  baseURL: '/api',
  timeout: 120000
})

request.interceptors.request.use((config) => {
  const token = getAuthToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code && res.code !== 200) {
      ElMessage.error(res.message || 'Request failed')
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      clearAuthToken()
      window.dispatchEvent(new CustomEvent('port-master:auth-required'))
      ElMessage.error('Authentication required')
      return Promise.reject(error)
    }
    ElMessage.error(error.response?.data?.message || error.message || 'Network error')
    return Promise.reject(error)
  }
)

export default request
