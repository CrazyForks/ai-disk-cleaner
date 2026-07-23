# AGENTS

## 基础规则

- 当你修改完代码后，必须执行 `pnpm lint` 和 `pnpm check` 清除所有 eslint 错误和警告

## UI 规则

- UI 必须使用 nextui 的组件，详见 [llms.txt](llms.txt)。样式同样必须使用 tailwindcss，禁止手搓 css。

## 表单规则

- 表单必须使用 react-hook-form 完成
- 所有 nextui 的表单组件，必须使用 `frontend/src/components/ControlledNextUIFormWrapper.tsx` 包裹，禁止自行适配
  `react-hook-form`，如果碰到不适配的组件，请停止当前任务并立即询问用户下一步
- 提交表单必须手动处理，禁止直接使用 form 的 onSubmit 事件

使用样例：

```typescript jsx
import ControlledNextUIFormWrapper from '@/components/ControlledNextUIFormWrapper'
import {Input, TextArea, Button} from '@heroui/react'
import { useForm } from 'react-hook-form'

type MyValues = {
    name: string
    description: string
}

const Foo: React.FC = () => {
    const {control, trigger, getValues} = useForm<MyValues>()
    
    const handleSubmit = async () => {
        if (!(await trigger())) {
            return
        }
        const values = getValues()
        // do real submit.
    }
    return (
        <div>
            <ControlledNextUIFormWrapper
                rules={{required: '请输入脚本名称'}}
                control={control}
                label="脚本名称"
                name="name"
            >
                <Input placeholder="请输入脚本名称"/>
            </ControlledNextUIFormWrapper>
            <ControlledNextUIFormWrapper
                control={control}
                label="描述"
                name="description"
            >
                <TextArea placeholder="补充这个脚本的用途、执行目标或运行注意事项。"/>
            </ControlledNextUIFormWrapper>
            <Button
                onPress={handleSubmit}
            >
                提交
            </Button>
        </div>
    )
}
```

## 错误处理

错误处理必须使用 toast 显示：

```typescript 
import {toastError} from "@/util/toast-error";

try {
    // do sth
} catch (e: unknown) {
    toastError(e, 'title')
}
```

## 用户确认

当你需要一个弹框来向用户寻求确认时，你应该使用 `showDialog` 方法，它在 `import { showDialog } from '@/components/DialogProvider'`

## 国际化

每次编写代码，必须提供国际化翻译，相关文件夹在 `frontend/src/i18n`，国际化 key 前缀尽量跟随路由。