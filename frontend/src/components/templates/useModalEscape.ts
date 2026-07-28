import { KeyboardEvent, useCallback } from 'react';

// Dialog-local Escape handling. Attaching this handler to the dialog root
// keeps nested modals safe: the topmost focused dialog consumes Escape and
// stops it before an underlying dialog can observe the same key press.
export function useModalEscape(onClose: () => void, disabled = false) {
    return useCallback((event: KeyboardEvent<HTMLElement>) => {
        if (event.key !== 'Escape') return;
        event.preventDefault();
        event.stopPropagation();
        if (!disabled) onClose();
    }, [disabled, onClose]);
}
