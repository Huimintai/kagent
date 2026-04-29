import { create } from 'zustand'
import { fetchOidcUser } from "./oidcUser";

interface UserStore {
  userId: string
  setUserId: (userId: string) => void
  clearLoginSession: () => void
  renewToken: () => void
}

const DEFAULT_USER_ID = 'admin@kagent.dev'
const USER_ID_KEY = 'kagent_user_id'
const OAUTH2_PROXY_SIGN_OUT_PATH = '/oauth2/sign_out?rd=/'


// Resolve user id from backend OIDC user when available, otherwise fallback to default.
const getInitialUserId = async (): Promise<string> => {
  if (typeof window === 'undefined') return DEFAULT_USER_ID;

  try {
    const data = await fetchOidcUser();
    if (data?.email) {
      localStorage.setItem(USER_ID_KEY, data.email);
      return data.email;
    }
  } catch {
    // Keep default value when OIDC request fails.
    localStorage.setItem(USER_ID_KEY, DEFAULT_USER_ID);
    return DEFAULT_USER_ID;
  }

  return DEFAULT_USER_ID;
}

export const useUserStore = create<UserStore>((set) => {
  const initialUserId =
    typeof window !== 'undefined'
      ? (localStorage.getItem(USER_ID_KEY) ?? DEFAULT_USER_ID)
      : DEFAULT_USER_ID

  if (typeof window !== 'undefined') {
    void getInitialUserId().then((userId) => {
      set({ userId })
    })
  }

  return {
    userId: initialUserId,
    setUserId: (userId: string) => {
      if (typeof window !== 'undefined') {
        localStorage.setItem(USER_ID_KEY, userId)
      }
      set({ userId })
    },
    clearLoginSession: () => {
      if (typeof window !== 'undefined') {
        localStorage.removeItem(USER_ID_KEY)
        window.location.assign(OAUTH2_PROXY_SIGN_OUT_PATH)
      }
      set({ userId: DEFAULT_USER_ID })
    },
    renewToken: () => {
      // Sign out then redirect to / — oauth2-proxy auto-starts a fresh OIDC
      // flow on the unauthenticated redirect target, issuing a new token.
      // We intentionally keep localStorage intact so userId survives the round-trip.
      if (typeof window !== 'undefined') {
        window.location.assign(OAUTH2_PROXY_SIGN_OUT_PATH)
      }
    },
  }
})