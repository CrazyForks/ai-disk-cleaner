import { useMemo } from 'react'
import { Card } from '@heroui/react'
import {
  ArcElement,
  BarController,
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  DoughnutController,
  LinearScale,
  Tooltip,
  type ChartOptions,
  type Plugin,
} from 'chart.js'
import { Bar, Doughnut } from 'react-chartjs-2'
import type { cleaningrecord } from '../../../../wailsjs/go/models'
import { formatBytes } from '@/hooks/useCleaningTask'
import { useTranslation } from 'react-i18next'

ChartJS.register(
  ArcElement,
  BarController,
  BarElement,
  CategoryScale,
  DoughnutController,
  LinearScale,
  Tooltip,
)

const fallbackUsageColors = [
  '#6366f1',
  '#14b8a6',
  '#f59e0b',
  '#f43f5e',
  '#8b5cf6',
]

type ScanResultStatsProps = {
  topUsages: cleaningrecord.DiskUsage[]
  trashFiles: cleaningrecord.TrashFile[]
}

type ChartTheme = {
  surface: string
  foreground: string
  muted: string
  separator: string
  overlay: string
  overlayForeground: string
  usageColors: string[]
  garbageColors: string[]
  fontFamily: string
}

export default function ScanResultStats({
  topUsages,
  trashFiles,
}: ScanResultStatsProps) {
  const { t } = useTranslation()
  const garbageSizes = trashFiles.reduce(
    (totals, file) => {
      totals[file.level] = (totals[file.level] ?? 0) + (file.size ?? 0)
      return totals
    },
    [0, 0, 0],
  )
  const theme = useMemo(readChartTheme, [])

  return (
    <Card className="rounded-3xl">
      <Card.Header>
        <Card.Title>{t('cleanup.detail.diskUsage')}</Card.Title>
        <Card.Description>{t('cleanup.detail.diskUsageDesc')}</Card.Description>
      </Card.Header>
      <Card.Content>
        <div className="grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,1.5fr)_minmax(18rem,1fr)]">
          <UsageDoughnutChart theme={theme} usages={topUsages} />
          <GarbageLevelChart sizes={garbageSizes} theme={theme} />
        </div>
        <div className="text-muted mt-5 space-y-2 text-sm">
          <div>{t('cleanup.detail.levelADesc')}</div>
          <div>{t('cleanup.detail.levelBDesc')}</div>
          <div>{t('cleanup.detail.levelCDesc')}</div>
        </div>
      </Card.Content>
    </Card>
  )
}

function UsageDoughnutChart({
  usages,
  theme,
}: {
  usages: cleaningrecord.DiskUsage[]
  theme: ChartTheme
}) {
  const { t } = useTranslation()
  const total = usages.reduce((sum, usage) => sum + Math.max(usage.size, 0), 0)
  const chartUsages = usages.filter((usage) => usage.size > 0)
  const hasData = chartUsages.length > 0
  const data = {
    labels: hasData
      ? chartUsages.map((usage) => usage.description || usage.path)
      : [t('cleanup.detail.noData')],
    datasets: [
      {
        data: hasData ? chartUsages.map((usage) => usage.size) : [1],
        backgroundColor: hasData
          ? chartUsages.map(
              (_, index) => theme.usageColors[index % theme.usageColors.length],
            )
          : [theme.separator],
        borderColor: theme.surface,
        borderWidth: hasData ? 3 : 0,
        hoverBorderColor: theme.surface,
        hoverOffset: hasData ? 6 : 0,
        spacing: hasData ? 1 : 0,
      },
    ],
  }
  const options: ChartOptions<'doughnut'> = {
    responsive: true,
    maintainAspectRatio: false,
    cutout: '67%',
    animation: { duration: 450 },
    plugins: {
      legend: { display: false },
      tooltip: {
        enabled: false,
        external: ({ chart, tooltip }) => {
          const parent = chart.canvas.parentElement
          if (!parent || !hasData) return

          let tooltipElement = parent.querySelector<HTMLDivElement>(
            '[data-usage-tooltip]',
          )
          if (!tooltipElement) {
            tooltipElement = document.createElement('div')
            tooltipElement.dataset.usageTooltip = ''
            Object.assign(tooltipElement.style, {
              position: 'absolute',
              zIndex: '50',
              pointerEvents: 'none',
              whiteSpace: 'nowrap',
              borderRadius: '12px',
              padding: '10px 12px',
              boxShadow: '0 8px 24px rgb(0 0 0 / 14%)',
              transition: 'opacity 120ms ease, transform 120ms ease',
            })
            parent.appendChild(tooltipElement)
          }

          if (tooltip.opacity === 0) {
            tooltipElement.style.opacity = '0'
            return
          }

          const title = document.createElement('div')
          title.textContent = tooltip.title[0] ?? ''
          title.style.fontWeight = '600'
          const value = document.createElement('div')
          value.textContent = t('cleanup.detail.used', {
            size: formatBytes(Number(tooltip.dataPoints[0]?.parsed ?? 0)),
          })
          value.style.marginTop = '3px'
          value.style.color = theme.muted
          tooltipElement.replaceChildren(title, value)
          tooltipElement.style.background = theme.overlay
          tooltipElement.style.color = theme.overlayForeground
          tooltipElement.style.fontFamily = theme.fontFamily
          tooltipElement.style.fontSize = '12px'
          tooltipElement.style.left = `${chart.canvas.offsetLeft + tooltip.caretX}px`
          tooltipElement.style.top = `${chart.canvas.offsetTop + tooltip.caretY}px`
          tooltipElement.style.opacity = '1'
          tooltipElement.style.transform = 'translate(-50%, calc(-100% - 10px))'
        },
        backgroundColor: theme.overlay,
        titleColor: theme.overlayForeground,
        bodyColor: theme.overlayForeground,
        titleFont: { family: theme.fontFamily, size: 12, weight: 600 },
        bodyFont: { family: theme.fontFamily, size: 12 },
        padding: 12,
        cornerRadius: 12,
        displayColors: true,
        boxPadding: 5,
      },
    },
  }
  const centerText = useMemo<Plugin<'doughnut'>>(
    () => ({
      id: 'usage-center-text',
      afterDraw: (chart) => {
        const { ctx, chartArea } = chart
        if (!chartArea) return

        const currentTotal =
          chart.data.labels?.[0] === t('cleanup.detail.noData')
            ? 0
            : (chart.data.datasets[0]?.data.reduce(
                (sum, value) => sum + (typeof value === 'number' ? value : 0),
                0,
              ) ?? 0)
        const centerX = (chartArea.left + chartArea.right) / 2
        const centerY = (chartArea.top + chartArea.bottom) / 2
        ctx.save()
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillStyle = theme.muted
        ctx.font = `12px ${theme.fontFamily}`
        ctx.fillText(t('cleanup.detail.total'), centerX, centerY - 11)
        ctx.fillStyle = theme.foreground
        ctx.font = `600 14px ${theme.fontFamily}`
        ctx.fillText(formatBytes(currentTotal), centerX, centerY + 12)
        ctx.restore()
      },
    }),
    [theme],
  )

  return (
    <div className="flex min-w-0 flex-col items-center gap-6 sm:flex-row sm:items-center">
      <div
        aria-label={`磁盘主要占用环形图，合计 ${formatBytes(total)}`}
        className="relative z-10 h-48 w-48 shrink-0 overflow-visible"
        role="img"
      >
        <Doughnut data={data} options={options} plugins={[centerText]} />
      </div>
      <div className="relative z-0 w-full min-w-0 flex-1 space-y-3">
        {usages.map((usage, index) => (
          <div
            key={`${usage.path}-${index}`}
            className="flex items-center gap-3"
          >
            <span
              className="size-2.5 shrink-0 rounded-full"
              style={{
                backgroundColor:
                  theme.usageColors[index % theme.usageColors.length],
              }}
            />
            <div className="min-w-0 flex-1">
              <p className="text-foreground truncate text-sm font-medium">
                {usage.description || usage.path}
              </p>
              <p className="text-muted truncate text-xs">{usage.path}</p>
            </div>
            <span className="text-muted shrink-0 text-xs">
              {formatBytes(usage.size)}
            </span>
          </div>
        ))}
        {usages.length === 0 && (
          <p className="text-muted text-sm">{t('cleanup.detail.noData')}</p>
        )}
      </div>
    </div>
  )
}

function GarbageLevelChart({
  sizes,
  theme,
}: {
  sizes: number[]
  theme: ChartTheme
}) {
  const { t } = useTranslation()
  const levels = [
    t('cleanup.detail.levelA'),
    t('cleanup.detail.levelB'),
    t('cleanup.detail.levelC'),
  ]
  const data = {
    labels: levels,
    datasets: [
      {
        data: sizes,
        backgroundColor: theme.garbageColors,
        borderColor: theme.garbageColors,
        borderWidth: 0,
        borderRadius: 10,
        borderSkipped: false as const,
        maxBarThickness: 42,
      },
    ],
  }
  const options: ChartOptions<'bar'> = {
    responsive: true,
    maintainAspectRatio: false,
    animation: { duration: 450 },
    layout: { padding: { top: 4 } },
    scales: {
      x: {
        grid: { display: false },
        border: { display: false },
        ticks: {
          color: theme.muted,
          font: { family: theme.fontFamily, size: 12, weight: 600 },
        },
      },
      y: {
        beginAtZero: true,
        grid: { color: theme.separator, drawTicks: false },
        border: { display: false },
        ticks: {
          color: theme.muted,
          padding: 8,
          maxTicksLimit: 4,
          font: { family: theme.fontFamily, size: 10 },
          callback: (value) => formatBytes(Number(value)),
        },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: theme.overlay,
        titleColor: theme.overlayForeground,
        bodyColor: theme.overlayForeground,
        titleFont: { family: theme.fontFamily, size: 12, weight: 600 },
        bodyFont: { family: theme.fontFamily, size: 12 },
        padding: 12,
        cornerRadius: 12,
        displayColors: false,
        callbacks: {
          label: (context) =>
            t('cleanup.detail.trashSize', {
              size: formatBytes(context.parsed.y ?? 0),
            }),
        },
      },
    },
  }

  return (
    <div className="bg-default/30 flex min-w-0 flex-col rounded-2xl p-4">
      <div className="mb-3 grid grid-cols-3 gap-2">
        {levels.map((level, index) => (
          <div key={level} className="min-w-0 text-center">
            <div className="mb-1 flex items-center justify-center gap-1.5">
              <span
                className="size-2 rounded-full"
                style={{ backgroundColor: theme.garbageColors[index] }}
              />
              <span className="text-muted text-xs">{level}</span>
            </div>
            <p className="text-foreground truncate text-sm font-semibold">
              {formatBytes(sizes[index] ?? 0)}
            </p>
          </div>
        ))}
      </div>
      <div
        aria-label={`垃圾风险等级柱状图，A 类 ${formatBytes(sizes[0] ?? 0)}，B 类 ${formatBytes(sizes[1] ?? 0)}，C 类 ${formatBytes(sizes[2] ?? 0)}`}
        className="min-h-44 flex-1"
        role="img"
      >
        <Bar data={data} options={options} />
      </div>
    </div>
  )
}

function readChartTheme(): ChartTheme {
  const color = (token: string, fallback: string) =>
    resolveCssColor(token, fallback)

  return {
    surface: color('--surface', '#ffffff'),
    foreground: color('--foreground', '#1f2937'),
    muted: color('--muted', '#6b7280'),
    separator: color('--separator', '#e5e7eb'),
    overlay: color('--overlay', '#ffffff'),
    overlayForeground: color('--overlay-foreground', '#1f2937'),
    usageColors: [
      color('--accent', fallbackUsageColors[0]),
      color('--success', fallbackUsageColors[1]),
      color('--warning', fallbackUsageColors[2]),
      color('--danger', fallbackUsageColors[3]),
      fallbackUsageColors[4],
    ],
    garbageColors: [
      color('--success', '#22c55e'),
      color('--warning', '#f59e0b'),
      color('--danger', '#ef4444'),
    ],
    fontFamily:
      getComputedStyle(document.body).fontFamily || 'Nunito, sans-serif',
  }
}

function resolveCssColor(token: string, fallback: string) {
  const probe = document.createElement('span')
  probe.style.color = `var(${token}, ${fallback})`
  probe.style.display = 'none'
  document.body.appendChild(probe)
  const resolved = getComputedStyle(probe).color
  probe.remove()
  return resolved || fallback
}
