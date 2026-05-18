export interface Message {
  id: number
  from_id: number
  to_id: number
  content: string
  is_read: boolean
  created_at: string
}

export interface SendMessageRequest {
  to_id: number
  content: string
}

export interface ListMessagesResponse {
  messages: Message[]
}
