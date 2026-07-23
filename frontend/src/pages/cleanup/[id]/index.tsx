import { Button, Card, Checkbox, Label } from '@heroui/react'
import type { Selection } from '@heroui/react'
import type { cleaner } from '../../../../wailsjs/go/models'
import { ArrowLeft } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import useCleaningTask from '@/hooks/useCleaningTask'
import { DeleteTrashFiles } from '../../../../wailsjs/go/main/App'
import LLMOutput from './LLMOutput'
import ScanProgressHeader from './ScanProgressHeader'
import ScanResultStats from './ScanResultStats'
import TrashFileTable from './TrashFileTable'
import DeleteFailuresDrawer from './DeleteFailuresDrawer'
import { showDialog } from '@/components/DialogProvider'
import ControlledNextUIFormWrapper from '@/components/ControlledNextUIFormWrapper'
import { useForm } from 'react-hook-form'
import { toastError } from '@/util/toast-error'
import { useTranslation } from 'react-i18next'

const runningStates = new Set(['SCANNING', 'ANALYZING'])

export default function CleanupDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams()
  const { error, isLoading, isStopping, records, refreshRecords, stop, task } =
    useCleaningTask()
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [deleteFailures, setDeleteFailures] = useState<cleaner.DeleteFailure[]>(
    [],
  )
  const [keepOriginalDirectoriesForRetry, setKeepOriginalDirectoriesForRetry] =
    useState(true)
  const recordID = Number(id)
  const currentTask = task?.id === recordID ? task : null
  const record = records.find((item) => item.id === recordID)
  const detail = currentTask ?? record
  const isRunning = currentTask ? runningStates.has(currentTask.state) : false
  const isAnalysisComplete = detail?.state === 'DONE'
  const trashFiles = record?.trashFiles ?? []

  useEffect(() => {
    setSelectedPaths(new Set())
    setDeleteFailures([])
  }, [recordID])

  const handleDelete = async (paths: string[] = Array.from(selectedPaths)) => {
    const deletablePaths = new Set(
      trashFiles.filter((file) => !file.isDeleted).map((file) => file.path),
    )
    const pathsToDelete = Array.from(new Set(paths)).filter((path) =>
      deletablePaths.has(path),
    )
    if (pathsToDelete.length === 0 || !record) {
      return
    }
    let keepOriginalDirectories = true
    showDialog({
      title: t('cleanup.detail.deleteDialogTitle'),
      message: (
        <DeleteConfirmation
          count={pathsToDelete.length}
          onKeepOriginalDirectoriesChange={(value) => {
            keepOriginalDirectories = value
          }}
        />
      ),
      color: 'danger',
      onConfirm: async () => {
        setIsDeleting(true)
        try {
          setDeleteFailures([])
          setKeepOriginalDirectoriesForRetry(keepOriginalDirectories)
          const failures = await DeleteTrashFiles(
            record.id,
            pathsToDelete,
            keepOriginalDirectories,
          )
          setDeleteFailures(failures)
          const failedPaths = failures.map((failure) => failure.path)
          const deletedPaths = new Set(
            pathsToDelete.filter((path) => {
              const candidatePath = resolveCandidatePath(record.path, path)
              return !failedPaths.some((failedPath) =>
                isSameOrDescendantPath(failedPath, candidatePath, record.path),
              )
            }),
          )
          setSelectedPaths(
            (current) =>
              new Set(
                Array.from(current).filter((path) => !deletedPaths.has(path)),
              ),
          )
          await refreshRecords()
        } catch (reason) {
          toastError(reason, t('cleanup.detail.deleteFailed'))
        } finally {
          setIsDeleting(false)
        }
      },
    })
  }

  const handleRetrySuccess = async (path: string) => {
    setDeleteFailures((current) =>
      current.filter((failure) => failure.path !== path),
    )
    setSelectedPaths(
      (current) =>
        new Set(
          Array.from(current).filter(
            (selectedPath) =>
              !isSameOrDescendantPath(
                path,
                resolveCandidatePath(record?.path ?? '', selectedPath),
                record?.path ?? '',
              ),
          ),
        ),
    )
    await refreshRecords()
  }

  const handleSelectionChange = (keys: Selection) => {
    const deletablePaths = new Set(
      trashFiles.filter((file) => !file.isDeleted).map((file) => file.path),
    )
    setSelectedPaths(
      keys === 'all'
        ? deletablePaths
        : new Set(
            Array.from(keys, String).filter((path) => deletablePaths.has(path)),
          ),
    )
  }

  return (
    <div className="space-y-6">
      <Button
        className="text-muted gap-2 rounded-xl"
        variant="ghost"
        onPress={() => navigate('/cleanup')}
      >
        <ArrowLeft aria-hidden="true" size={16} />
        {t('cleanup.detail.backToRecords')}
      </Button>

      {detail ? (
        <>
          <ScanProgressHeader
            detail={detail}
            isRunning={isRunning}
            isStopping={isStopping || Boolean(currentTask?.stopping)}
            scanProgress={currentTask?.scanProgress}
            onStop={() => void stop()}
          />

          {record && isAnalysisComplete && (
            <>
              <ScanResultStats
                topUsages={record.topUsages ?? []}
                trashFiles={trashFiles}
              />
              <TrashFileTable
                files={trashFiles}
                isDeleting={isDeleting}
                rootPath={record.path}
                selectedPaths={selectedPaths}
                onDelete={(paths) => void handleDelete(paths)}
                onSelectionChange={handleSelectionChange}
              />
            </>
          )}

          <LLMOutput isRunning={isRunning} output={detail.llmOutput} />
        </>
      ) : (
        <Card className="rounded-3xl">
          <Card.Content className="text-muted flex min-h-56 items-center justify-center text-sm">
            {isLoading ? '正在加载清理任务…' : '没有找到这条清理任务'}
          </Card.Content>
        </Card>
      )}

      <DeleteFailuresDrawer
        failures={deleteFailures}
        keepOriginalDirectories={keepOriginalDirectoriesForRetry}
        recordID={recordID}
        onClose={() => setDeleteFailures([])}
        onRetrySuccess={handleRetrySuccess}
      />

      {error && <p className="text-danger text-sm">{error}</p>}
    </div>
  )
}

function resolveCandidatePath(rootPath: string, candidatePath: string) {
  if (
    candidatePath.startsWith('/') ||
    /^[A-Za-z]:[\\/]/.test(candidatePath) ||
    /^[\\/]{2}/.test(candidatePath)
  ) {
    return candidatePath
  }
  const separator = rootPath.includes('\\') ? '\\' : '/'
  return `${rootPath.replace(/[\\/]+$/, '')}${separator}${candidatePath.replace(
    /^[\\/]+/,
    '',
  )}`
}

function comparablePath(path: string, rootPath: string) {
  const normalized = path.replace(/\\/g, '/').replace(/\/{2,}/g, '/')
  return /^[A-Za-z]:[\\/]/.test(rootPath)
    ? normalized.toLowerCase()
    : normalized
}

function isSameOrDescendantPath(
  path: string,
  parentPath: string,
  rootPath: string,
) {
  const comparable = comparablePath(path, rootPath).replace(/\/+$/, '')
  const comparableParent = comparablePath(parentPath, rootPath).replace(
    /\/+$/,
    '',
  )
  return (
    comparable === comparableParent ||
    comparable.startsWith(`${comparableParent}/`)
  )
}

type DeleteConfirmationProps = {
  count: number
  onKeepOriginalDirectoriesChange: (value: boolean) => void
}

type DeleteConfirmationValues = {
  keepOriginalDirectories: boolean
}

function DeleteConfirmation({
  count,
  onKeepOriginalDirectoriesChange,
}: DeleteConfirmationProps) {
  const { t } = useTranslation()
  const { control } = useForm<DeleteConfirmationValues>({
    defaultValues: { keepOriginalDirectories: true },
  })

  return (
    <div className="space-y-4">
      <p>{t('cleanup.detail.deleteConfirmation', { count })}</p>
      <ControlledNextUIFormWrapper
        control={control}
        name="keepOriginalDirectories"
      >
        {(field) => (
          <Checkbox
            isSelected={field.value}
            onChange={(value) => {
              field.onChange(value)
              onKeepOriginalDirectoriesChange(value)
            }}
          >
            <Checkbox.Content>
              <Checkbox.Control>
                <Checkbox.Indicator />
              </Checkbox.Control>
              <Label>{t('cleanup.detail.keepOriginalDirectories')}</Label>
            </Checkbox.Content>
          </Checkbox>
        )}
      </ControlledNextUIFormWrapper>
    </div>
  )
}
