import {useState} from 'react';
import {RunAuditSave} from '../../wailsjs/go/main/App';
import {vm} from '../../wailsjs/go/models';
import {RISK_INFO, RiskKey, CONFIDENCE_STYLE, Confidence} from '../data/riskInfo';
import {RiskInfoIcon} from './RiskInfoIcon';

interface Props {
    selectedChar: number;
    activeSlots: boolean[];
}

const TIER_LABEL: Record<number, string> = {
    0: 'Tier 0',
    1: 'Tier 1',
    2: 'Tier 2',
};

const TIER_CLASS: Record<number, string> = {
    0: 'bg-zinc-500/20 text-zinc-300 border-zinc-500/40',
    1: 'bg-yellow-500/20 text-yellow-200 border-yellow-500/40',
    2: 'bg-red-500/20 text-red-200 border-red-500/40',
};

export function AuditPanel({selectedChar, activeSlots}: Props) {
    const [slotIdx, setSlotIdx] = useState<number>(selectedChar);
    const [report, setReport] = useState<vm.AuditReport | null>(null);
    const [running, setRunning] = useState(false);
    const [error, setError] = useState<string>('');

    const handleRun = async () => {
        if (!activeSlots[slotIdx]) {
            setError(`Slot ${slotIdx + 1} is empty.`);
            setReport(null);
            return;
        }
        setRunning(true);
        setError('');
        try {
            const r = await RunAuditSave(slotIdx);
            setReport(r);
        } catch (e) {
            setError(String(e));
            setReport(null);
        } finally {
            setRunning(false);
        }
    };

    return (
        <div className="card px-4 py-3 space-y-3">
            <div className="flex items-start justify-between gap-3">
                <div className="flex-1 space-y-1">
                    <span className="text-[10px] font-black uppercase tracking-widest text-foreground block">Save Audit</span>
                    <p className="text-[10px] text-muted-foreground leading-relaxed">
                        <strong>Deterministic check.</strong> Scans the selected slot for offline-detectable ban markers
                        (caps, cut content flags, NG+ scaling). <strong>Server-side rules are unknown — passing the audit is not a guarantee
                        the save will not be flagged online.</strong>
                    </p>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <select
                    value={slotIdx}
                    onChange={e => setSlotIdx(Number(e.target.value))}
                    className="flex-1 bg-background border border-border/50 rounded px-2.5 py-1.5 text-[11px] font-mono focus:outline-none focus:ring-1 focus:ring-primary/20"
                    disabled={running}
                >
                    {Array.from({length: 10}, (_, i) => (
                        <option key={i} value={i} disabled={!activeSlots[i]}>
                            Slot {i + 1}{activeSlots[i] ? '' : ' (empty)'}
                        </option>
                    ))}
                </select>
                <button
                    onClick={handleRun}
                    disabled={running}
                    className="px-3 py-1.5 rounded bg-primary text-primary-foreground text-[10px] font-black uppercase tracking-widest hover:brightness-110 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                >
                    {running ? 'Running…' : 'Run Audit'}
                </button>
            </div>

            {error && (
                <p className="text-[10px] text-red-400 px-2 py-1 rounded bg-red-500/10 border border-red-500/30">{error}</p>
            )}

            {report && <ReportView report={report} />}
        </div>
    );
}

function ReportView({report}: {report: vm.AuditReport}) {
    const issues = report.issues || [];
    const passed = report.passedChecks;
    const total = report.totalChecks;

    if (issues.length === 0) {
        return (
            <div className="rounded border border-emerald-500/30 bg-emerald-500/5 p-3 space-y-1">
                <p className="text-[10px] font-black uppercase tracking-widest text-emerald-400">
                    No deterministic ban markers detected
                </p>
                <p className="text-[10px] text-muted-foreground leading-relaxed">
                    {passed}/{total} checks passed. <strong>Server-side rules are unknown — this is not a guarantee.</strong>
                    {' '}Last edit may still trigger detection on first online sync.
                </p>
            </div>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-[10px] font-black uppercase tracking-widest text-foreground">
                {issues.length} issue{issues.length === 1 ? '' : 's'} · {passed}/{total} checks passed
            </p>
            <p className="text-[10px] text-muted-foreground">
                Server-side rules are unknown — fixing all issues is not a safety guarantee.
            </p>
            <ul className="space-y-1.5">
                {issues.map((iss, i) => (
                    <IssueRow key={i} issue={iss} />
                ))}
            </ul>
        </div>
    );
}

function IssueRow({issue}: {issue: vm.AuditIssue}) {
    const tier = issue.severity;
    const confidence = issue.confidence as Confidence;
    const conf = CONFIDENCE_STYLE[confidence] || CONFIDENCE_STYLE.speculated;
    const knownRiskKey = issue.riskKey && (issue.riskKey in RISK_INFO);

    return (
        <li className="rounded border border-border/50 bg-muted/10 p-2 space-y-1">
            <div className="flex items-center gap-1.5 flex-wrap">
                <span className={`text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border ${TIER_CLASS[tier]}`}>
                    {TIER_LABEL[tier]}
                </span>
                <span className={`text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border ${conf.classes}`}>
                    {conf.label}
                </span>
                {knownRiskKey && <RiskInfoIcon riskKey={issue.riskKey as RiskKey} />}
                <span className="text-[10px] font-bold text-foreground">{issue.field}</span>
            </div>
            <p className="text-[10px] text-muted-foreground leading-relaxed">
                {issue.message}
            </p>
            {issue.mitigation && (
                <p className="text-[10px] text-foreground/80 leading-relaxed">
                    <span className="font-black uppercase tracking-widest text-[8px] text-muted-foreground/70 mr-1">Fix:</span>
                    {issue.mitigation}
                </p>
            )}
        </li>
    );
}
