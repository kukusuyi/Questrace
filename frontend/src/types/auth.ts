export interface RegisterPayload {
  education_stage: string
  username: string
  password: string
  email: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface AuthResponse {
  user_id: number
  username: string
  token: string
}
