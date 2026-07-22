import { Button, Card, Checkbox, Label } from '@heroui/react'
import type { Selection } from '@heroui/react'
import { ArrowLeft } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import useCleaningTask from '@/hooks/useCleaningTask'
import { DeleteTrashFiles } from '../../../../wailsjs/go/main/App'
import LLMOutput from './LLMOutput'
import ScanProgressHeader from './ScanProgressHeader'
import ScanResultStats from './ScanResultStats'
import TrashFileTable from './TrashFileTable'
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
  const recordID = Number(id)
  const currentTask = task?.id === recordID ? task : null
  const record = records.find((item) => item.id === recordID)
  const detail = currentTask ?? record
  const isRunning = currentTask ? runningStates.has(currentTask.state) : false
  const isAnalysisComplete = detail?.state === 'DONE'
  const trashFiles = record?.trashFiles ?? []

  useEffect(() => {
    setSelectedPaths(new Set())
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
      title: '删除文件',
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
          await DeleteTrashFiles(
            record.id,
            pathsToDelete,
            keepOriginalDirectories,
          )
          const deletedPaths = new Set(pathsToDelete)
          setSelectedPaths(
            (current) =>
              new Set(
                Array.from(current).filter((path) => !deletedPaths.has(path)),
              ),
          )
          await refreshRecords()
        } catch (reason) {
          toastError(reason, '删除失败')
        } finally {
          setIsDeleting(false)
        }
      },
    })
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

      {error && <p className="text-danger text-sm">{error}</p>}
    </div>
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
  const { control } = useForm<DeleteConfirmationValues>({
    defaultValues: { keepOriginalDirectories: true },
  })

  return (
    <div className="space-y-4">
      <p>确定删除选中的 {count} 个文件或目录吗？此操作无法撤销。</p>
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
              <Label>保留原始目录</Label>
            </Checkbox.Content>
          </Checkbox>
        )}
      </ControlledNextUIFormWrapper>
    </div>
  )
}
