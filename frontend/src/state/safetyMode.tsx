import {createContext, useCallback, useContext, useState, ReactNode} from 'react';

// Online Safety Mode — global toggle that gates Tier 1/Tier 2 edits across the UI.
//   Tier 0 (cosmetic / read-only)    → always allowed
//   Tier 1 (caution: bulk unlocks)   → forced confirmation modal when enabled
//   Tier 2 (high risk: cut content,  → disabled when enabled
//          stat overflows, etc.)
// Phase 2 only ships the toggle, banner, and hook API; Tier 1/2 controls start
// consuming `isDisabledFor` / `requireConfirmFor` in Phases 3-5.

export type RiskTier = 0 | 1 | 2;

export interface SafetyModeContextValue {
    enabled: boolean;
    setEnabled: (value: boolean) => void;
    isDisabledFor: (tier: RiskTier) => boolean;
    requireConfirmFor: (tier: RiskTier) => boolean;
}

const SafetyModeContext = createContext<SafetyModeContextValue | null>(null);
const STORAGE_KEY = 'setting:onlineSafetyMode';

export function SafetyModeProvider({children}: {children: ReactNode}) {
    const [enabled, setEnabledState] = useState<boolean>(() => {
        try {
            return localStorage.getItem(STORAGE_KEY) === 'true';
        } catch {
            return false;
        }
    });

    const setEnabled = useCallback((value: boolean) => {
        try {
            localStorage.setItem(STORAGE_KEY, String(value));
        } catch {
            // localStorage unavailable — keep in-memory state only
        }
        setEnabledState(value);
    }, []);

    const isDisabledFor = useCallback(
        (tier: RiskTier) => enabled && tier === 2,
        [enabled],
    );

    const requireConfirmFor = useCallback(
        (tier: RiskTier) => enabled && tier >= 1,
        [enabled],
    );

    return (
        <SafetyModeContext.Provider value={{enabled, setEnabled, isDisabledFor, requireConfirmFor}}>
            {children}
        </SafetyModeContext.Provider>
    );
}

export function useSafetyMode(): SafetyModeContextValue {
    const ctx = useContext(SafetyModeContext);
    if (!ctx) {
        throw new Error('useSafetyMode must be used within a SafetyModeProvider');
    }
    return ctx;
}
