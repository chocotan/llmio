"use client"

import { Bar, BarChart, CartesianGrid, LabelList, XAxis, YAxis } from "recharts"

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
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"

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

export interface RankingChartItem {
  label: string
  value: number
}

// 根据模型数据生成图表配置
const generateChartConfig = (data: RankingChartItem[], valueLabel: string) => {
  const config: ChartConfig = {
    value: {
      label: valueLabel,
    },
  }

  data.forEach((item, index) => {
    config[item.label] = {
      label: item.label,
      color: predefinedColors[index % predefinedColors.length],
    }
  })

  return config
}

// 根据模型数据生成图表数据
const generateChartData = (data: RankingChartItem[]) => {
  return data.map((item, index) => ({
    label: item.label,
    value: item.value,
    fill: predefinedColors[index % predefinedColors.length],
  }))
}

interface ModelRankingChartProps {
  data: RankingChartItem[]
  title: string
  description?: string
  valueLabel: string
}

export function ModelRankingChart({ data, title, description, valueLabel }: ModelRankingChartProps) {
  const chartData = generateChartData(data)
  const chartConfig = generateChartConfig(data, valueLabel)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
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
              dataKey="label"
              tickLine={false}
              axisLine={false}
              tickMargin={16}
              interval={0}
              tickFormatter={(value) => String(value)}
            />
            <YAxis
              dataKey="value"
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
              dataKey="value"
              fill="var(--color-value)"
              radius={[8, 8, 0, 0]}
            >
              <LabelList
                dataKey="value"
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
