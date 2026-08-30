import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactNode, RefObject } from "react";
import { Button } from "../Button/Button";
import { actions, body, content, description, overlay, title } from "./Dialog.css";

export type DialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  closeLabel: ReactNode;
  children: ReactNode;
  returnFocusRef?: RefObject<HTMLElement | null>;
};

/**
 * The single modal implementation. Radix owns focus trapping, Escape, the
 * interaction lock and focus restoration; features only provide content.
 */
export function Dialog({
  open,
  onOpenChange,
  title: heading,
  description: hint,
  closeLabel,
  children,
  returnFocusRef,
}: DialogProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className={overlay} />
        <RadixDialog.Content
          className={content}
          onCloseAutoFocus={(event) => {
            if (!returnFocusRef?.current) return;
            event.preventDefault();
            returnFocusRef.current.focus();
          }}
        >
          <RadixDialog.Title className={title}>{heading}</RadixDialog.Title>
          {hint ? (
            <RadixDialog.Description className={description}>{hint}</RadixDialog.Description>
          ) : null}
          <div className={body}>{children}</div>
          <div className={actions}>
            <RadixDialog.Close asChild>
              <Button>{closeLabel}</Button>
            </RadixDialog.Close>
          </div>
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}
