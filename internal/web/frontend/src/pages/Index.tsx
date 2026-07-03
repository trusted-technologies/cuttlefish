import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { fetchSlaves, type SlaveInfo } from "@/lib/api"
import { ModeToggle } from "@/components/mode-toggle"

export default function IndexPage() {
  const [slaves, setSlaves] = useState<SlaveInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchSlaves()
      .then((data) => {
        setSlaves(data)
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-tight">Cuttlefish</h1>
          <ModeToggle />
        </div>
      </header>

      <main className="flex-1 max-w-7xl mx-auto px-4 py-8 w-full">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-3xl font-bold">Network nodes</h2>
          <span className="text-sm text-muted-foreground">
            {slaves.length} registered
          </span>
        </div>

        {loading && <p className="text-muted-foreground">Loading...</p>}
        {error && <p className="text-destructive">{error}</p>}

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {slaves.map((slave) => (
            <Link key={slave.id} to={`/slave/${slave.id}`}>
              <Card className="hover:bg-accent/50 transition-colors h-full">
                <CardHeader className="relative">
                  <span className="absolute top-4 right-4 inline-flex h-2.5 w-2.5 rounded-full bg-green-500" />
                  <CardTitle>{slave.name}</CardTitle>
                  <p className="text-sm text-muted-foreground">
                    {slave.location}
                  </p>
                </CardHeader>
                <CardContent>
                  <div className="space-y-1 text-sm">
                    {slave.ipv4 && (
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">IPv4</span>
                        <code className="font-mono">{slave.ipv4}</code>
                      </div>
                    )}
                    {slave.ipv6 && (
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">IPv6</span>
                        <code className="font-mono break-all">{slave.ipv6}</code>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        {!loading && slaves.length === 0 && (
          <Card className="p-8 text-center text-muted-foreground border-dashed">
            No slaves registered yet.
          </Card>
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
