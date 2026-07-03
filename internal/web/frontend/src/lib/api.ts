import type { SlaveInfo, MyIPInfo, TestFile } from "@/types"

export async function fetchSlaves(): Promise<SlaveInfo[]> {
  const res = await fetch("/api/slaves")
  if (!res.ok) throw new Error("Failed to fetch slaves")
  return res.json()
}

export async function fetchMyIP(): Promise<MyIPInfo> {
  const res = await fetch("/api/myip")
  if (!res.ok) throw new Error("Failed to fetch my IP")
  return res.json()
}

export function runToolStream(
  slaveId: string,
  tool: string,
  target: string,
  ipv6: boolean,
  onLine: (line: string) => void,
  onError: (err: string) => void,
  onDone: () => void
): EventSource {
  const params = new URLSearchParams({ target, count: "4", timeout: "30" })
  if (ipv6) params.set("ipv6", "true")
  const es = new EventSource(`/api/slaves/${slaveId}/${tool}?${params}`)
  es.addEventListener("result", (e) => {
    const data = JSON.parse(e.data)
    if (data.line) onLine(data.line)
    if (data.error) onError(data.error)
    if (data.done) {
      es.close()
      onDone()
    }
  })
  es.onerror = () => {
    es.close()
    onDone()
  }
  return es
}

export function statsStream(
  slaveId: string,
  onPoint: (point: { ts: string; iface: string; rx: number; tx: number }) => void,
  onError?: () => void
): EventSource {
  const es = new EventSource(`/api/slaves/${slaveId}/stats`)
  es.addEventListener("stats", (e) => {
    onPoint(JSON.parse(e.data))
  })
  es.onerror = () => {
    es.close()
    onError?.()
  }
  return es
}

export { type SlaveInfo, type MyIPInfo, type TestFile }
