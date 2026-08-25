import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import { Menu as BaseMenu } from "@base-ui/react/menu";
import { Popover as BasePopover } from "@base-ui/react/popover";
import { Select as BaseSelect } from "@base-ui/react/select";
import { Tabs as BaseTabs } from "@base-ui/react/tabs";
import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ButtonHTMLAttributes, HTMLAttributes, ReactNode } from "react";

export function Button({ className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`ui-button ${className}`.trim()} {...props} />;
}

export function IconButton({ "aria-label": label, className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  if (!label) throw new Error("IconButton requires aria-label");
  return <button aria-label={label} className={`ui-icon-button ${className}`.trim()} {...props} />;
}

export function Badge({ className = "", ...props }: HTMLAttributes<HTMLSpanElement>) {
  return <span className={`ui-badge ${className}`.trim()} {...props} />;
}

export function Separator({ className = "", ...props }: HTMLAttributes<HTMLHRElement>) {
  return <hr className={`ui-separator ${className}`.trim()} {...props} />;
}

export function Spinner({ label, className = "" }: { label: string; className?: string }) {
  return <span aria-label={label} className={`ui-spinner ${className}`.trim()} role="status" />;
}

type MessageProps = HTMLAttributes<HTMLDivElement> & { title: string; children?: ReactNode };

export function EmptyState({ title, children, className = "", ...props }: MessageProps) {
  return <div className={`ui-empty-state ${className}`.trim()} data-ui-state="empty" {...props}><strong>{title}</strong>{children}</div>;
}

export function Diagnostic({ title, children, className = "", ...props }: MessageProps) {
  return <div className={`ui-diagnostic ${className}`.trim()} data-ui-state="error" role="alert" {...props}><strong>{title}</strong>{children}</div>;
}

export const Dialog = BaseDialog;
export const Tabs = BaseTabs;
export const Tooltip = BaseTooltip;
export const Popover = BasePopover;
export const Menu = BaseMenu;
export const Select = BaseSelect;
