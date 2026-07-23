import { Button, Drawer, Table } from '@heroui/react'
import { RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { cleaner } from '../../../../wailsjs/go/models'
import { DeleteTrashFiles } from '../../../../wailsjs/go/main/App'
import { toastError } from '@/util/toast-error'

type DeleteFailuresDialogProps = {
  failures: cleaner.DeleteFailure[]
  keepOriginalDirectories: boolean
  recordID: number
  onClose: () => void
  onRetrySuccess: (path: string) => Promise<void>
}

export default function DeleteFailuresDrawer({
  failures,
  keepOriginalDirectories,
  recordID,
  onClose,
  onRetrySuccess,
}: DeleteFailuresDialogProps) {
  const { t } = useTranslation()
  const [retryingPath, setRetryingPath] = useState<string | null>(null)

  const handleRetry = async (path: string) => {
    setRetryingPath(path)
    try {
      const retryFailures = await DeleteTrashFiles(
        recordID,
        [path],
        keepOriginalDirectories,
      )
      if (retryFailures.length > 0) {
        toastError(retryFailures[0].message, t('cleanup.detail.retryFailed'))
        return
      }
      await onRetrySuccess(path)
    } catch (reason) {
      toastError(reason, t('cleanup.detail.retryFailed'))
    } finally {
      setRetryingPath(null)
    }
  }

  return (
    <Drawer.Backdrop
      isOpen={failures.length > 0}
      onOpenChange={(isOpen) => {
        if (!isOpen) onClose()
      }}
    >
      <Drawer.Content placement="bottom">
        <Drawer.Dialog>
          <Drawer.CloseTrigger />
          <Drawer.Handle />
          <Drawer.Header>
            {/*<Drawer.Icon status="danger" />*/}
            <Drawer.Heading>
              {t('cleanup.detail.deleteFailuresTitle')}
            </Drawer.Heading>
          </Drawer.Header>
          <Drawer.Body>
            <Table variant="secondary">
              <Table.ScrollContainer>
                <Table.Content
                  aria-label={t('cleanup.detail.deleteFailuresTableLabel')}
                >
                  <Table.Header>
                    <Table.Column isRowHeader>{t('common.path')}</Table.Column>
                    <Table.Column>
                      {t('cleanup.detail.failureReason')}
                    </Table.Column>
                    <Table.Column>{t('common.actions')}</Table.Column>
                  </Table.Header>
                  <Table.Body>
                    {failures.map((failure) => (
                      <Table.Row key={failure.path} id={failure.path}>
                        <Table.Cell>
                          <span className="block max-w-96 break-all">
                            {failure.path}
                          </span>
                        </Table.Cell>
                        <Table.Cell>
                          <span className="text-danger block max-w-96 break-all">
                            {failure.message}
                          </span>
                        </Table.Cell>
                        <Table.Cell>
                          <Button
                            className="gap-2"
                            isDisabled={
                              retryingPath !== null &&
                              retryingPath !== failure.path
                            }
                            isPending={retryingPath === failure.path}
                            size="sm"
                            variant="secondary"
                            onPress={() => void handleRetry(failure.path)}
                          >
                            <RotateCcw aria-hidden="true" size={14} />
                            {t('cleanup.detail.retry')}
                          </Button>
                        </Table.Cell>
                      </Table.Row>
                    ))}
                  </Table.Body>
                </Table.Content>
              </Table.ScrollContainer>
            </Table>
          </Drawer.Body>
          <Drawer.Footer>
            <Button slot="close" variant="ghost" onPress={onClose}>
              {t('common.close')}
            </Button>
          </Drawer.Footer>
        </Drawer.Dialog>
      </Drawer.Content>
    </Drawer.Backdrop>
  )
}
