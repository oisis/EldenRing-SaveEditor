import {useSafetyMode} from '../state/safetyMode';

export function SafetyModeBanner() {
    const {enabled} = useSafetyMode();
    if (!enabled) return null;
    return (
        <div className="flex-shrink-0 bg-amber-600/95 text-amber-50 px-4 py-1.5 flex items-center justify-center gap-3 shadow-md border-b border-amber-700/40">
            <span className="text-base leading-none">⚠</span>
            <span className="text-[10px] font-black uppercase tracking-[0.15em] text-center">
                Online Safety Mode — Tier 2 edits disabled, Tier 1 requires confirmation
            </span>
        </div>
    );
}
