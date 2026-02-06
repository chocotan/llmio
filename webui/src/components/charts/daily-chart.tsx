"use client"

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import type { DailyMetric } from "@/lib/api"

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

interface DailyChartProps {
  data: DailyMetric[]
  title?: string
  description?: string
}

export function DailyChart({ data, title = "每日统计", description = "请求和Token使用趋势" }: DailyChartProps) {
  // Format chart data
  const chartData = data.map(item => ({
    date: item.date,
    reqs: item.reqs,
    tokens: Math.round(item.tokens / TOKEN_DISPLAY_DIVISOR), // Convert to K tokens for better display
  }))

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
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
              dataKey="date"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              tickFormatter={(value) => {
                // Parse date string as local date to avoid timezone issues
                const [year, month, day] = value.split('-').map(Number)
                const date = new Date(year, month - 1, day)
                return `${date.getMonth() + 1}/${date.getDate()}`
              }}
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
                  labelFormatter={(value) => {
                    // Parse date string as local date to avoid timezone issues
                    const [year, month, day] = value.split('-').map(Number)
                    const date = new Date(year, month - 1, day)
                    return date.toLocaleDateString("zh-CN", {
                      month: "short",
                      day: "numeric",
                    })
                  }}
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
      </CardContent>
    </Card>
  )
}
