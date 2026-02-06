"use client"

import { useState, useEffect } from "react"
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { getDailyMetrics, getHourlyMetrics } from "@/lib/api"
import { toast } from "sonner"

const chartConfig = {
  reqs: {
    label: "请求数",
    color: "hsl(var(--chart-1))",
  },
  tokens: {
    label: "Token数",
    color: "hsl(var(--chart-2))",
  },
} satisfies ChartConfig

const TOKEN_DISPLAY_DIVISOR = 1000 // Convert tokens to K for better readability

type TimeRange = "7d" | "30d" | "90d"
type Granularity = "hour" | "day"

interface MetricsChartProps {
  title?: string
  description?: string
}

export function MetricsChart({ title = "统计图表", description = "请求和Token使用趋势" }: MetricsChartProps) {
  const [timeRange, setTimeRange] = useState<TimeRange>("30d")
  const [granularity, setGranularity] = useState<Granularity>("day")
  const [loading, setLoading] = useState(false)
  const [chartData, setChartData] = useState<Array<{
    time: string
    reqs: number
    tokens: number
  }>>([])

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        if (granularity === "day") {
          const days = timeRange === "7d" ? 7 : timeRange === "30d" ? 30 : 90
          const data = await getDailyMetrics(days)
          setChartData(data.map(item => ({
            time: item.date,
            reqs: item.reqs,
            tokens: Math.round(item.tokens / TOKEN_DISPLAY_DIVISOR),
          })))
        } else {
          // For hourly view, use hours based on time range
          const hours = timeRange === "7d" ? 168 : timeRange === "30d" ? 720 : 2160 // 7*24, 30*24, 90*24
          const data = await getHourlyMetrics(hours)
          setChartData(data.map(item => ({
            time: item.hour,
            reqs: item.reqs,
            tokens: Math.round(item.tokens / TOKEN_DISPLAY_DIVISOR),
          })))
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        toast.error(`获取统计数据失败: ${message}`)
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    void fetchData()
  }, [timeRange, granularity])

  const formatXAxisTick = (value: string) => {
    if (granularity === "day") {
      // Parse date string as local date to avoid timezone issues
      const [year, month, day] = value.split('-').map(Number)
      const date = new Date(year, month - 1, day)
      return `${date.getMonth() + 1}/${date.getDate()}`
    } else {
      // Format hour: "2024-01-15 08:00:00" -> "01/15 08:00"
      const parts = value.split(' ')
      if (parts.length >= 2) {
        const [year, month, day] = parts[0].split('-').map(Number)
        const date = new Date(year, month - 1, day)
        const time = parts[1].substring(0, 5) // "HH:MM"
        return `${date.getMonth() + 1}/${date.getDate()} ${time}`
      }
      return value
    }
  }

  const formatTooltipLabel = (value: string) => {
    if (granularity === "day") {
      const [year, month, day] = value.split('-').map(Number)
      const date = new Date(year, month - 1, day)
      return date.toLocaleDateString("zh-CN", {
        year: "numeric",
        month: "short",
        day: "numeric",
      })
    } else {
      // Format hour for tooltip
      const parts = value.split(' ')
      if (parts.length >= 2) {
        const [year, month, day] = parts[0].split('-').map(Number)
        const date = new Date(year, month - 1, day)
        return `${date.toLocaleDateString("zh-CN", {
          month: "short",
          day: "numeric",
        })} ${parts[1].substring(0, 5)}`
      }
      return value
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </div>
          <div className="flex gap-2">
            <Select value={timeRange} onValueChange={(value) => setTimeRange(value as TimeRange)}>
              <SelectTrigger className="w-[120px]">
                <SelectValue placeholder="选择时间范围" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7d">最近7天</SelectItem>
                <SelectItem value="30d">最近30天</SelectItem>
                <SelectItem value="90d">最近90天</SelectItem>
              </SelectContent>
            </Select>
            <Select value={granularity} onValueChange={(value) => setGranularity(value as Granularity)}>
              <SelectTrigger className="w-[120px]">
                <SelectValue placeholder="选择粒度" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="hour">按小时</SelectItem>
                <SelectItem value="day">按天</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex h-[320px] items-center justify-center">
            <div className="text-muted-foreground">加载中...</div>
          </div>
        ) : (
          <ChartContainer config={chartConfig} className="aspect-auto h-[320px] w-full">
            <BarChart
              accessibilityLayer
              data={chartData}
              margin={{
                left: 12,
                right: 12,
              }}
            >
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="time"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tickFormatter={formatXAxisTick}
                interval="preserveStartEnd"
              />
              <YAxis
                yAxisId="left"
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) => Number(value).toLocaleString()}
                width={60}
                label={{ value: '调用次数', angle: -90, position: 'insideLeft', style: { textAnchor: 'middle' } }}
              />
              <YAxis
                yAxisId="right"
                orientation="right"
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) => `${value}K`}
                width={60}
                label={{ value: 'Token数量 (K)', angle: 90, position: 'insideRight', style: { textAnchor: 'middle' } }}
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    labelFormatter={formatTooltipLabel}
                    formatter={(value, name) => {
                      if (name === "tokens") {
                        return [`${value}K`, chartConfig[name as keyof typeof chartConfig]?.label || name]
                      }
                      return [value, chartConfig[name as keyof typeof chartConfig]?.label || name]
                    }}
                  />
                }
              />
              <ChartLegend content={<ChartLegendContent />} />
              <Bar
                yAxisId="left"
                dataKey="reqs"
                fill="var(--color-reqs)"
                radius={[4, 4, 0, 0]}
              />
              <Bar
                yAxisId="right"
                dataKey="tokens"
                fill="var(--color-tokens)"
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
