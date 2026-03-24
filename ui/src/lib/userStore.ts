import { create } from 'zustand'

interface UserStore {
  userId: string
  initialized: boolean
  setUserId: (userId: string) => void
}

const DEFAULT_USER_ID = 'admin@kagent.dev'
const USER_ID_KEY = 'kagent_user_id'

// Get initial state from localStorage if available
const getInitialUserId = () => {
  if (typeof window === 'undefined') return DEFAULT_USER_ID
  return localStorage.getItem(USER_ID_KEY) || DEFAULT_USER_ID
}

export const useUserStore = create<UserStore>((set) => ({
  userId: getInitialUserId(),
  initialized: false,
  setUserId: (userId: string) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(USER_ID_KEY, userId)
    }
    set({ userId, initialized: true })
  }
}))