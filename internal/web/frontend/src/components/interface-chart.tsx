import { useMemo } from "react"
import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export interface StatsPoint {
  ts: string
  iface: string
  rx: number
  tx: number
}

interface InterfaceChartProps {
  data: StatsPoint[]
  iface: string
}

function formatBytes(bps: number): string {
  if (bps === 0) return "0 b/s"
  const units = ["b/s", "Kb/s", "Mb/s", "Gb/s"]
  const i = Math.floor(Math.log10(bps) / 3)
  return `${(bps / Math.pow(1000, i)).toFixed(1)} ${units[i]}`
}

export function InterfaceChart({ data, iface }: InterfaceChartProps) {
  const chartData = useMemo(() => {
    return data.map((p) => ({
      time: new Date(p.ts).toLocaleTimeString(),
      rx: p.rx,
      tx: p.tx,
    }))
  }, [data])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{iface}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id={`rx-${iface}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#22c55e" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                </linearGradient>
                <linearGradient id={`tx-${iface}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="time" tick={{ fontSize: 12 }} />
              <YAxis tickFormatter={formatBytes} tick={{ fontSize: 12 }} />
              <Tooltip
                formatter={(value: any) =>
                  [formatBytes(Number(value) || 0), ""]
                }
              />
              <Legend />
              <Area
                type="monotone"
                dataKey="rx"
                name="RX"
                stroke="#22c55e"
                fill={`url(#rx-${iface})`}
                isAnimationActive={false}
              />
              <Area
                type="monotone"
                dataKey="tx"
                name="TX"
                stroke="#3b82f6"
                fill={`url(#tx-${iface})`}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
