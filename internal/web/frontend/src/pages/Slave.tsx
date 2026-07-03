import { useEffect, useRef, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { ArrowLeft, Copy } from "lucide-react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ModeToggle } from "@/components/mode-toggle"
import { InterfaceChart, type StatsPoint } from "@/components/interface-chart"
import { fetchSlaves, fetchMyIP, runToolStream, statsStream, type SlaveInfo } from "@/lib/api"

function copyText(text: string) {
  navigator.clipboard.writeText(text)
}

function getPort(publicUrl: string) {
  try {
    return new URL(publicUrl).port || "5201"
  } catch {
    return "5201"
  }
}

export default function SlavePage() {
  const { id } = useParams<{ id: string }>()
  const [slave, setSlave] = useState<SlaveInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [target, setTarget] = useState("")
  const [ipv6, setIpv6] = useState(false)
  const [output, setOutput] = useState("")
  const outputRef = useRef<HTMLPreElement>(null)

  const [myip, setMyip] = useState<{ ipv4?: string; ipv6?: string }>({})

  const [stats, setStats] = useState<Record<string, StatsPoint[]>>({})

  useEffect(() => {
    fetchSlaves()
      .then((list) => {
        const s = list.find((x) => x.id === id)
        if (!s) throw new Error("Slave not found")
        setSlave(s)
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [id])

  useEffect(() => {
    fetchMyIP().then(setMyip).catch(() => {})
  }, [])

  useEffect(() => {
    if (!slave) return
    const es = statsStream(slave.id, (point) => {
      setStats((prev) => {
        const list = [...(prev[point.iface] || []), point]
        if (list.length > 60) list.shift()
        return { ...prev, [point.iface]: list }
      })
    })
    return () => es.close()
  }, [slave])

  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight
    }
  }, [output])

  const runTool = (tool: string) => {
    if (!target) return
    setOutput(`Running ${tool} ${target}...\n`)
    runToolStream(
      slave!.id,
      tool,
      target,
      ipv6,
      (line) => setOutput((o) => o + line),
      (err) => setOutput((o) => o + "ERROR: " + err + "\n"),
      () => {}
    )
  }

  const iperfPort = slave?.iperf_port || getPort(slave?.public_url || "")

  if (loading) return <p className="p-8">Loading...</p>
  if (error) return <p className="p-8 text-destructive">{error}</p>
  if (!slave) return null

  const files = (slave.file_sizes || ["1M", "10M", "100M", "1G", "10G", "100G"]).map(
    (size) => ({
      name: `${size}.bin`,
      size,
      url: `${slave.public_url}/files/${size}`,
    })
  )

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/" className="text-2xl font-bold tracking-tight">
            Cuttlefish
          </Link>
          <ModeToggle />
        </div>
      </header>

      <main className="flex-1 max-w-7xl mx-auto px-4 py-8 w-full space-y-6">
        <Link
          to="/"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="w-4 h-4 mr-1" />
          Back
        </Link>

        <Card>
          <CardHeader>
            <CardTitle>{slave.name}</CardTitle>
            <CardDescription>{slave.location}</CardDescription>
          </CardHeader>
          <CardContent>
            <div
              className={`grid gap-4 ${
                slave.ipv6 ? "md:grid-cols-2" : "grid-cols-1"
              }`}
            >
              <div className="p-4 bg-muted rounded-lg">
                <p className="text-sm text-muted-foreground">IPv4</p>
                <div className="flex items-center justify-between gap-3 mt-2">
                  <code className="font-mono text-lg break-all">
                    {slave.ipv4}
                  </code>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => copyText(slave.ipv4)}
                  >
                    <Copy className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              {slave.ipv6 && (
                <div className="p-4 bg-muted rounded-lg">
                  <p className="text-sm text-muted-foreground">IPv6</p>
                  <div className="flex items-center justify-between gap-3 mt-2">
                    <code className="font-mono text-lg break-all">
                      {slave.ipv6}
                    </code>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyText(slave.ipv6)}
                    >
                      <Copy className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <div className="grid gap-6 lg:grid-cols-2 items-start">
          <Card>
            <CardHeader>
              <CardTitle>Network tools</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-3">
                <Input
                  placeholder="example.com"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  className="flex-1 min-w-[12rem]"
                />
                {slave.ipv6 && (
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={ipv6}
                      onChange={(e) => setIpv6(e.target.checked)}
                      className="w-4 h-4"
                    />
                    IPv6
                  </label>
                )}
              </div>
              <div className="flex flex-wrap gap-2">
                <Button onClick={() => runTool("ping")}>Ping</Button>
                <Button onClick={() => runTool("mtr")}>MTR</Button>
                <Button onClick={() => runTool("traceroute")}>
                  Traceroute
                </Button>
              </div>
              <pre
                ref={outputRef}
                className="h-80 overflow-auto rounded-lg bg-black text-green-400 p-4 text-sm font-mono whitespace-pre-wrap"
              >
                {output}
              </pre>
            </CardContent>
          </Card>

          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Speed test files</CardTitle>
                <CardDescription>
                  Download test files served directly by this slave.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-3">
                  {files.map((f) => (
                    <a
                      key={f.name}
                      href={f.url}
                      className="flex items-center justify-between p-3 bg-muted rounded-lg hover:bg-accent transition-colors"
                    >
                      <span className="font-mono text-sm">{f.name}</span>
                      <span className="text-xs text-muted-foreground">
                        {f.size}
                      </span>
                    </a>
                  ))}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>iPerf3 traffic</CardTitle>
                <CardDescription>
                  Run these from your machine against this node.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {slave.ipv4 && (
                  <>
                    <CopyCommand
                      label="IPv4 incoming"
                      cmd={`iperf3 -4 -c ${slave.ipv4} -p ${iperfPort} -P 4`}
                    />
                    <CopyCommand
                      label="IPv4 outgoing"
                      cmd={`iperf3 -4 -c ${slave.ipv4} -p ${iperfPort} -P 4 -R`}
                    />
                  </>
                )}
                {slave.ipv6 && (
                  <>
                    <CopyCommand
                      label="IPv6 incoming"
                      cmd={`iperf3 -6 -c ${slave.ipv6} -p ${iperfPort} -P 4`}
                    />
                    <CopyCommand
                      label="IPv6 outgoing"
                      cmd={`iperf3 -6 -c ${slave.ipv6} -p ${iperfPort} -P 4 -R`}
                    />
                  </>
                )}
              </CardContent>
            </Card>

            {(myip.ipv4 || myip.ipv6) && (
              <Card>
                <CardHeader>
                  <CardTitle>Your IP</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  {myip.ipv4 && (
                    <p>
                      <span className="text-muted-foreground">IPv4</span>{" "}
                      <code className="font-mono">{myip.ipv4}</code>
                    </p>
                  )}
                  {myip.ipv6 && (
                    <p>
                      <span className="text-muted-foreground">IPv6</span>{" "}
                      <code className="font-mono">{myip.ipv6}</code>
                    </p>
                  )}
                </CardContent>
              </Card>
            )}
          </div>
        </div>

        {Object.keys(stats).length > 0 && (
          <div className="space-y-4">
            <h2 className="text-2xl font-bold">Interface traffic</h2>
            <div className="grid gap-4 lg:grid-cols-2">
              {Object.entries(stats).map(([iface, data]) => (
                <InterfaceChart key={iface} iface={iface} data={data} />
              ))}
            </div>
          </div>
        )}
      </main>

      <footer className="border-t">
        <div className="max-w-7xl mx-auto px-4 py-6 text-sm text-muted-foreground">
          Open-source Looking Glass ·{" "}
          <a
            href="https://github.com/trusted-technologies/cuttlefish"
            className="underline hover:text-foreground"
          >
            GitHub
          </a>
        </div>
      </footer>
    </div>
  )
}

function CopyCommand({ label, cmd }: { label: string; cmd: string }) {
  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="flex items-center justify-between gap-3 p-3 bg-muted rounded-lg">
        <code className="font-mono text-sm break-all">{cmd}</code>
        <Button variant="outline" size="sm" onClick={() => copyText(cmd)}>
          <Copy className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}
