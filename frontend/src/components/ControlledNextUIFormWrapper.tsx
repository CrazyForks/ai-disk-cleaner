import type {
  Control,
  FieldValues,
  ControllerRenderProps,
  ControllerFieldState,
  Path,
  PathValue,
  RegisterOptions,
} from 'react-hook-form'
import { Controller } from 'react-hook-form'
import { Description, FieldError, Label, TextField } from '@heroui/react'
import React from 'react'
import i18n from 'i18next'

interface ControlledNextUIInputProps<
  Values extends FieldValues,
  MyPath extends Path<Values> = Path<Values>,
> {
  label?: string
  control: Control<Values>
  name: MyPath
  defaultValue?: PathValue<Values, Path<Values>>
  rules?: Omit<
    RegisterOptions<Values, MyPath>,
    'valueAsNumber' | 'valueAsDate' | 'setValueAs' | 'disabled'
  >
  description?: string
  children:
    | React.ReactNode
    | ((
        field: ControllerRenderProps<Values, MyPath>,
        fieldState: ControllerFieldState,
      ) => React.ReactNode)
}

type StringSupplier = () => string
const DEFAULT_VALIDATION_MESSAGE_SUPPLIERS: Record<
  string,
  StringSupplier | undefined
> = {
  required: () => i18n.t('validation.notEmpty'),
}

function ensureErrorMessage(type: string, msg?: string): string {
  if (!msg || msg.length === 0) {
    const supplier = DEFAULT_VALIDATION_MESSAGE_SUPPLIERS[type]
    if (supplier) {
      return supplier()
    }
  }
  return msg ?? ''
}

function ControlledNextUIFormWrapper<Values extends FieldValues>({
  control,
  name,
  label,
  children,
  defaultValue,
  description,
  rules,
}: ControlledNextUIInputProps<Values>) {
  return (
    <Controller
      control={control}
      name={name}
      rules={rules}
      defaultValue={defaultValue}
      render={({ field, fieldState }) => {
        return (
          <TextField
            {...field}
            isRequired={!!rules?.required}
            isInvalid={fieldState.invalid}
          >
            {label && <Label>{label}</Label>}
            {typeof children === 'function'
              ? children(field, fieldState)
              : children}
            {fieldState.error ? (
              <FieldError>
                {ensureErrorMessage(
                  fieldState.error.type,
                  fieldState.error.message,
                )}
              </FieldError>
            ) : (
              <Description>{description}</Description>
            )}
          </TextField>
        )
      }}
    />
  )
}

export default ControlledNextUIFormWrapper
