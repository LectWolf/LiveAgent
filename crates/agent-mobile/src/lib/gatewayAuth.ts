import { getGatewayOrigin } from "@/lib/gatewayOrigin";
import { nativeFetch as fetch } from "@/lib/nativeNet";

function readUnauthorizedErrorMessage(errorText: string) {
  return errorText === "unauthorized" ? "Access Token 错误，请检查后重试。" : errorText;
}

async function readFetchError(response: Response, fallback: string) {
  const raw = (await response.text()).trim();
  if (!raw) {
    return fallback;
  }

  try {
    const payload = JSON.parse(raw) as { error?: unknown; message?: unknown };
    const errorText =
      typeof payload.error === "string"
        ? payload.error.trim()
        : typeof payload.message === "string"
          ? payload.message.trim()
          : "";
    return readUnauthorizedErrorMessage(errorText || raw);
  } catch {
    return readUnauthorizedErrorMessage(raw);
  }
}

const AGENT_ID_PATTERN =
  /^agent-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export function looksLikeAgentID(value: string) {
  return AGENT_ID_PATTERN.test(value.trim());
}

export function normalizeGatewayAccessToken(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }

  const match = trimmed.match(/^bearer\s+(.+)$/i);
  if (!match) {
    return trimmed;
  }

  return match[1]?.trim() ?? "";
}

export async function verifyGatewayAccessToken(input: string) {
  const token = normalizeGatewayAccessToken(input);
  if (!token) {
    throw new Error("请输入 Access Token。");
  }

  const response = await fetch(`${getGatewayOrigin()}/api/status`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(await readFetchError(response, "Access Token 验证失败。"));
  }

  return token;
}
