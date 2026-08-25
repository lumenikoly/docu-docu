import { createRoot } from "react-dom/client";
import { Badge, Button, Diagnostic, Dialog, EmptyState, IconButton, Menu, Popover, Select, Separator, Spinner, Tabs, Tooltip } from "../ui";
import "../styles/portal.css";
import "../styles/ui.css";

function Gallery() {
  return <main className="ui-gallery">
    <h1>Toudocu UI</h1>
    <section><h2>Основы</h2><div className="ui-row"><Button>Сохранить</Button><IconButton aria-label="Закрыть">×</IconButton><Badge>Готово</Badge><Spinner label="Загрузка" /></div></section>
    <Separator />
    <section><h2>Состояния</h2><EmptyState title="Здесь пока пусто" /><Diagnostic title="Не удалось загрузить">Повторите попытку.</Diagnostic></section>
    <section><h2>Диалог и подсказки</h2><div className="ui-row">
      <Dialog.Root><Dialog.Trigger className="ui-button">Открыть диалог</Dialog.Trigger><Dialog.Portal><Dialog.Backdrop className="ui-backdrop" /><Dialog.Viewport className="ui-dialog-viewport"><Dialog.Popup className="ui-dialog"><Dialog.Title>Подтверждение</Dialog.Title><Dialog.Description>Проверьте действие.</Dialog.Description><Dialog.Close className="ui-button">Закрыть</Dialog.Close></Dialog.Popup></Dialog.Viewport></Dialog.Portal></Dialog.Root>
      <Tooltip.Provider><Tooltip.Root><Tooltip.Trigger className="ui-button">Подсказка</Tooltip.Trigger><Tooltip.Portal><Tooltip.Positioner sideOffset={8}><Tooltip.Popup className="ui-tooltip">Полезная деталь</Tooltip.Popup></Tooltip.Positioner></Tooltip.Portal></Tooltip.Root></Tooltip.Provider>
      <Popover.Root><Popover.Trigger className="ui-button">Подробнее</Popover.Trigger><Popover.Portal><Popover.Positioner sideOffset={8}><Popover.Popup className="ui-popup"><Popover.Title>Контекст</Popover.Title><Popover.Description>Дополнительные сведения.</Popover.Description></Popover.Popup></Popover.Positioner></Popover.Portal></Popover.Root>
    </div></section>
    <section><h2>Выбор</h2><Tabs.Root defaultValue="first"><Tabs.List aria-label="Разделы примера" className="ui-tabs"><Tabs.Tab value="first">Первое</Tabs.Tab><Tabs.Tab value="second">Второе</Tabs.Tab></Tabs.List><Tabs.Panel value="first">Первая вкладка</Tabs.Panel><Tabs.Panel value="second">Вторая вкладка</Tabs.Panel></Tabs.Root><div className="ui-row">
      <Menu.Root><Menu.Trigger className="ui-button">Действия</Menu.Trigger><Menu.Portal><Menu.Positioner sideOffset={8}><Menu.Popup className="ui-popup"><Menu.Item>Открыть</Menu.Item><Menu.Item>Скопировать</Menu.Item></Menu.Popup></Menu.Positioner></Menu.Portal></Menu.Root>
      <Select.Root defaultValue="ready"><Select.Label>Статус</Select.Label><Select.Trigger className="ui-button"><Select.Value /></Select.Trigger><Select.Portal><Select.Positioner sideOffset={8}><Select.Popup className="ui-popup"><Select.List><Select.Item value="ready"><Select.ItemIndicator>✓</Select.ItemIndicator><Select.ItemText>Готово</Select.ItemText></Select.Item><Select.Item value="draft"><Select.ItemIndicator>✓</Select.ItemIndicator><Select.ItemText>Черновик</Select.ItemText></Select.Item></Select.List></Select.Popup></Select.Positioner></Select.Portal></Select.Root>
    </div></section>
  </main>;
}

const root = document.querySelector("#ui-root");
if (root) createRoot(root).render(<Gallery />);
