import {
  StartGameResponse,
  ApiStartGameResponse,
  DescribeResponse,
  ApiDescribeResponse,
  ChatResponse,
  ApiChatRequest,
  ApiChatResponse,
  WorldReadyResponse,
  ApiWorldReadyResponse,
} from "./api.types";
import { v4 as uuidv4 } from "uuid";
import { GET, POST } from "./api.service";

export async function StartGame(
  sessionId?: string,
): Promise<StartGameResponse> {
  const sessionUUID = sessionId ?? uuidv4();

  const response = await POST<ApiStartGameResponse>(`startgame/${sessionId}`);
  return {
    success: response.status == 200,
    error: response.data.error,
    ready: response.data.ready || false,
    sessionUUID: sessionUUID,
  };
}

export async function DescribeGame(
  sessionUUID: string,
): Promise<DescribeResponse> {
  const response = await GET<ApiDescribeResponse>(`describe/${sessionUUID}`);
  return response.data;
}

export async function Chat(
  sessionUUID: string,
  reqBody: ApiChatRequest,
): Promise<ChatResponse> {
  const response = await POST<ApiChatResponse>(`chat/${sessionUUID}`, reqBody);
  return response.data;
}

export async function WorldReady(
  sessionUUID: string,
): Promise<WorldReadyResponse> {
  const response = await GET<ApiWorldReadyResponse>(`chat/${sessionUUID}`);
  return {
    ready: response.status == 200,
  };
}
