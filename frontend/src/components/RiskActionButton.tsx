import {ReactNode, useState} from 'react';
import {createPortal} from 'react-dom';
import {RISK_INFO, RiskKey, RiskLevel} from '../data/riskInfo';
import {RiskInfoIcon} from './RiskInfoIcon';
import {useSafetyMode} from '../state/safetyMode';

interface Props {
    riskKey?: RiskKey | null;
    onConfirm: () => void;
    children: ReactNode;
    className?: string;
    disabled?: boolean;
    title?: string;
}

const DOTS_COLOR: Record<RiskLevel, string> = {
    low: 'text-yellow-400',
    medium: 'text-orange-400',
    high: 'text-red-400',
};

const DOTS: Record<RiskLevel, string> = {
    low: '● ○ ○',
    medium: '● ● ○',
    high: '● ● ●',
};

const BORDER_COLOR: Record<RiskLevel, string> = {
    low: 'border-yellow-500/40 shadow-yellow-500/10',
    medium: 'border-orange-500/50 shadow-orange-500/15',
    high: 'border-red-500/60 shadow-red-500/20',
};

export function RiskActionButton({riskKey, onConfirm, children, className, disabled, title}: Props) {
    const [showModal, setShowModal] = useState(false);
    const safetyMode = useSafetyMode();
    const entry = riskKey ? RISK_INFO[riskKey] : null;
    // Modal is gated entirely on Online Safety Mode — when it's off, action runs immediately.
    // The ⚠ info icon next to the button stays as the always-available educational affordance.
    const requiresConfirm = !!entry && safetyMode.requireConfirmFor(entry.tier);

    const handleClick = () => {
        if (disabled) return;
        if (requiresConfirm) {
            setShowModal(true);
        } else {
            onConfirm();
        }
    };

    return (
        <>
            <span className="inline-flex items-center gap-1.5">
                <button
                    type="button"
                    onClick={handleClick}
                    disabled={disabled}
                    title={title}
                    className={className}
                >
                    {children}
                </button>
                {entry && <RiskInfoIcon riskKey={riskKey!} />}
            </span>
            {showModal && entry && riskKey && (
                <RiskActionModal
                    entry={entry}
                    onCancel={() => setShowModal(false)}
                    onProceed={() => {
                        setShowModal(false);
                        onConfirm();
                    }}
                />
            )}
        </>
    );
}

interface ModalProps {
    entry: typeof RISK_INFO[RiskKey];
    onCancel: () => void;
    onProceed: () => void;
}

function RiskActionModal({entry, onCancel, onProceed}: ModalProps) {
    return createPortal(
        <div
            className="fixed inset-0 z-[150] flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-200"
            onClick={onCancel}
        >
            <div
                className={`bg-card p-6 rounded-2xl border-2 w-full max-w-md mx-4 shadow-2xl space-y-4 animate-in zoom-in-95 duration-200 ${BORDER_COLOR[entry.level]}`}
                onClick={e => e.stopPropagation()}
            >
                <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-3">
                        <span className={`text-2xl leading-none ${DOTS_COLOR[entry.level]}`}>⚠</span>
                        <h3 className="text-sm font-black uppercase tracking-[0.15em] text-foreground">{entry.title}</h3>
                    </div>
                    <span className={`text-[10px] font-mono whitespace-nowrap ${DOTS_COLOR[entry.level]}`} title={`Risk level: ${entry.level}`}>
                        {DOTS[entry.level]}
                    </span>
                </div>

                <div className="space-y-3 text-[11px] leading-relaxed text-muted-foreground">
                    <ModalSection heading="Why is this flagged?" body={entry.whyBan} />
                    <ModalSection heading="Community reports" body={entry.reports} />
                    <ModalSection heading="How to mitigate" body={entry.mitigation} />
                </div>

                <p className="text-[9px] font-bold uppercase tracking-widest text-amber-400 px-1">
                    Online Safety Mode is on — confirmation required.
                </p>

                <div className="flex gap-2 pt-1">
                    <button
                        onClick={onCancel}
                        className="flex-1 px-4 py-2.5 bg-muted/30 text-muted-foreground rounded-md text-[10px] font-black uppercase tracking-widest border border-border hover:bg-muted/50 transition-all"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={onProceed}
                        className="flex-1 px-4 py-2.5 bg-foreground text-background rounded-md text-[10px] font-black uppercase tracking-widest hover:brightness-110 active:scale-95 transition-all"
                    >
                        Proceed
                    </button>
                </div>
            </div>
        </div>,
        document.body,
    );
}

function ModalSection({heading, body}: {heading: string; body: string}) {
    return (
        <div>
            <p className="text-[9px] font-black uppercase tracking-[0.15em] text-foreground/80 mb-1">{heading}</p>
            <p>{body}</p>
        </div>
    );
}
