// SDK 内 naive-ui 按需注册：独立 Vue 应用各自安装。
// 只登记实际使用的组件，命名导入保证 tree-shaking；
// 新增 n-* 标签时在此补登
import type { App } from 'vue'
import {
  NButton,
  NCard,
  NCheckbox,
  NCheckboxGroup,
  NDataTable,
  NDatePicker,
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NDrawer,
  NDrawerContent,
  NEmpty,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NGi,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioButton,
  NRadioGroup,
  NRate,
  NSelect,
  NSlider,
  NSpin,
  NSwitch,
} from 'naive-ui'

const components = [
  NButton,
  NCard,
  NCheckbox,
  NCheckboxGroup,
  NDataTable,
  NDatePicker,
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NDrawer,
  NDrawerContent,
  NEmpty,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NGi,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioButton,
  NRadioGroup,
  NRate,
  NSelect,
  NSlider,
  NSpin,
  NSwitch,
]

export const naivePlugin = {
  install(app: App) {
    // 注册名与 naive-ui 自带 install 同构：前缀 N + 组件 name；
    // 组件 alias（不带 N）同步注册，如 FormItemGridItem 的别名 FormItemGi
    components.forEach((component) => {
      const { name, alias } = component as { name?: string; alias?: string[] }
      if (name) app.component(`N${name}`, component)
      if (Array.isArray(alias)) {
        alias.forEach((aliasName) => app.component(`N${aliasName}`, component))
      }
    })
  },
}
