export interface SlaveInfo {
  id: string
  name: string
  public_url: string
  ipv4: string
  ipv6: string
  location: string
  iperf_port: string
  file_sizes?: string[]
  last_seen: string
}

export interface MyIPInfo {
  ipv4: string
  ipv6: string
}

export interface TestFile {
  name: string
  size: string
  url: string
}

export interface StatsPoint {
  ts: string
  iface: string
  rx: number
  tx: number
}
