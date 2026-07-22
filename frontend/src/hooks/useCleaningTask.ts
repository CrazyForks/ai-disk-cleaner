import { useCallback, useEffect, useState } from 'react'
import {
  GetActiveCleaning,
  ListCleaningRecords,
  StopCleaning,
} from '../../wailsjs/go/main/App'
import { cleaner } from '../../wailsjs/go/models'
import type { cleaningrecord } from '../../wailsjs/go/models'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import i18n from 'i18next'

const TASK_UPDATED_EVENT = 'cleaning:task-updated'
const LLM_DELTA_EVENT = 'cleaning:llm-delta'
const terminalStates = new Set(['DONE', 'ERROR', 'CANCELLED'])

type LLMDelta = {
  recordId: number
  delta: string
}

const hasWailsRuntime = () =>
  typeof window !== 'undefined' && 'go' in window && 'runtime' in window

export default function useCleaningTask() {
  const [task, setTask] = useState<cleaner.CleaningTaskSnapshot | null>(null)
  const [records, setRecords] = useState<cleaningrecord.CleaningRecord[]>([])
  const [error, setError] = useState('')
  const [isStopping, setIsStopping] = useState(false)
  const [isLoading, setIsLoading] = useState(hasWailsRuntime)

  const refreshRecords = useCallback(async () => {
    if (!hasWailsRuntime()) {
      return
    }
    try {
      const result = await ListCleaningRecords()
      setRecords(result ?? [])
    } catch (reason) {
      setError(toErrorMessage(reason))
    }
  }, [])

  useEffect(() => {
    if (!hasWailsRuntime()) {
      return
    }

    let mounted = true
    void (async () => {
      try {
        const result = await GetActiveCleaning()
        if (mounted) {
          setTask(
            result ? cleaner.CleaningTaskSnapshot.createFrom(result) : null,
          )
        }
        await refreshRecords()
      } catch (reason) {
        if (mounted) {
          setError(toErrorMessage(reason))
        }
      } finally {
        if (mounted) {
          setIsLoading(false)
        }
      }
    })()

    const stopTaskListener = EventsOn(
      TASK_UPDATED_EVENT,
      (payload: unknown) => {
        if (!mounted) {
          return
        }
        const snapshot = cleaner.CleaningTaskSnapshot.createFrom(payload)
        setTask(snapshot)
        setIsStopping(snapshot.stopping)
        if (terminalStates.has(snapshot.state)) {
          void refreshRecords()
        }
      },
    )
    const stopDeltaListener = EventsOn(LLM_DELTA_EVENT, (payload: LLMDelta) => {
      if (!mounted || !payload?.delta) {
        return
      }
      setTask((current) => {
        if (!current || current.id !== payload.recordId) {
          return current
        }
        return cleaner.CleaningTaskSnapshot.createFrom({
          ...current,
          llmOutput: `${current.llmOutput ?? ''}${payload.delta}`,
        })
      })
    })

    return () => {
      mounted = false
      stopTaskListener()
      stopDeltaListener()
    }
  }, [refreshRecords])

  const stop = useCallback(async () => {
    if (!task || isStopping) {
      return
    }
    setIsStopping(true)
    setError('')
    try {
      await StopCleaning(task.id)
    } catch (reason) {
      setIsStopping(false)
      setError(toErrorMessage(reason))
    }
  }, [isStopping, task])

  return {
    task,
    records,
    error,
    isStopping,
    isLoading,
    refreshRecords,
    stop,
  }
}

export function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  )
  const value = bytes / 1024 ** unitIndex
  return `${value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`
}

export function formatDate(value: unknown) {
  const date = new Date(String(value))
  if (Number.isNaN(date.getTime())) {
    return 'Unknown Time'
  }
  return new Intl.DateTimeFormat(i18n.language.replaceAll('_', '-'), {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function stateLabel(state: string) {
  switch (state) {
    case 'SCANNING':
      return i18n.t('common.scanning')
    case 'ANALYZING':
      return i18n.t('common.analyzing')
    case 'DONE':
      return i18n.t('common.done')
    case 'CANCELLED':
      return i18n.t('common.cancelled')
    case 'ERROR':
      return i18n.t('common.error')
    default:
      return state
  }
}

function toErrorMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : String(reason)
}
