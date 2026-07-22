import type { ConfirmDialogRef, DialogConfig } from './ConfirmDialog'
import ConfirmDialog from './ConfirmDialog'
import React, { useEffect, useRef } from 'react'
import { getDialogQueue } from './dialog-queue'

const DialogProvider: React.FC = () => {
  const dialog = useRef<ConfirmDialogRef>(null)

  useEffect(() => {
    getDialogQueue().addElementAddListener((e) => {
      dialog.current?.showDialog(e)
    })
  }, [])

  return <ConfirmDialog ref={dialog} />
}

export const showDialog = (config: DialogConfig) => {
  getDialogQueue().addElement(config)
}

export default DialogProvider
