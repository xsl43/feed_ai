import api from './client'
import type { Message, SendMessageRequest, ListMessagesResponse } from '../types'

export const messageAPI = {
  send: (data: SendMessageRequest) => api.post<Message>('/message/send', data),
  list: (peer_id: number) => api.post<ListMessagesResponse>('/message/list', { peer_id }),
}
