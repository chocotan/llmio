"use client"

import { Bar, BarChart, CartesianGrid, LabelList, XAxis, YAxis } from "recharts"

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import type { ModelTokenUsage } from "@/lib/api"

// 预定义颜色数组，按顺序生成颜色
const predefinedColors = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
  "var(--chart-6)",
  "var(--chart-7)",
  "var(--chart-8)",
  "var(--chart-9)",
  "var(--chart-10)",
]

// 根据模型数据生成图表配置
const generateChartConfig = (data: ModelTokenUsage[]) => {
  const config: ChartConfig = {
    tokens: {
      label: "Tokens",
    },
  }

  data.forEach((item, index) => {
    config[item.model] = {
      label: item.model,
      color: predefinedColors[index % predefinedColors.length],
    }
  })

  return config
}

// 根据模型数据生成图表数据
const generateChartData = (data: ModelTokenUsage[]) => {
  return data.map((item, index) => ({
    model: item.model,
    tokens: item.tokens,
    fill: predefinedColors[index % predefinedColors.length],
  }))
}

interface ModelRankingChartProps {
  data: ModelTokenUsage[]
  title: string
  description?: string
}

export function ModelRankingChart({ data, title, description }: ModelRankingChartProps) {
  const chartData = generateChartData(data)
  const chartConfig = generateChartConfig(data)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="aspect-auto h-[320px] w-full">
          <BarChart
            accessibilityLayer
            data={chartData}
            barSize={32}
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="model"
              tickLine={false}
              axisLine={false}
              tickMargin={16}
              interval={0}
              tickFormatter={(value) => String(value)}
            />
            <YAxis
              dataKey="tokens"
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => Number(value).toLocaleString()}
              width={80}
            />
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent indicator="line" hideLabel />}
            />
            <Bar
              dataKey="tokens"
              fill="var(--color-tokens)"
              radius={[8, 8, 0, 0]}
            >
              <LabelList
                dataKey="tokens"
                position="top"
                offset={12}
                className="fill-foreground font-medium"
                fontSize={12}
              />
            </Bar>
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
