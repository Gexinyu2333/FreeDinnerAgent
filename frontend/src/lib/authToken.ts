const accessTokenKey = "freedinner.accessToken";
const refreshTokenKey = "freedinner.refreshToken";

export function getAccessToken() {
  return window.localStorage.getItem(accessTokenKey);
}

export function setTokens(accessToken: string, refreshToken?: string) {
  window.localStorage.setItem(accessTokenKey, accessToken);
  if (refreshToken) {
    window.localStorage.setItem(refreshTokenKey, refreshToken);
  }
}

export function clearTokens() {
  window.localStorage.removeItem(accessTokenKey);
  window.localStorage.removeItem(refreshTokenKey);
}
