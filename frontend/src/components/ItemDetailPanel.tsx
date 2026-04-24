import {useState} from 'react';
import {db} from '../../wailsjs/go/models';

interface ItemDetailPanelProps {
    item: db.ItemEntry;
    onClose: () => void;
}

export function ItemDetailPanel({item, onClose}: ItemDetailPanelProps) {
    const [brokenIcon, setBrokenIcon] = useState(false);

    return (
        <div className="h-full flex flex-col border-l border-border bg-card overflow-hidden">
            {/* Header */}
            <div className="bg-card/95 backdrop-blur-md border-b border-border p-4 flex items-start gap-3 shrink-0">
                <div className="w-14 h-14 rounded-lg bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden shrink-0">
                    {brokenIcon ? (
                        <span className="text-xl font-black text-muted-foreground/30">?</span>
                    ) : (
                        <img src={item.iconPath} alt="" className="w-10 h-10 object-contain drop-shadow-md" onError={() => setBrokenIcon(true)} />
                    )}
                </div>
                <div className="flex-1 min-w-0">
                    <h3 className="text-[11px] font-black uppercase tracking-widest text-foreground truncate">{item.name}</h3>
                    <p className="text-[8px] font-bold text-muted-foreground uppercase tracking-widest mt-0.5">
                        {item.category.replace(/_/g, ' ')}
                    </p>
                    <p className="text-[8px] font-mono text-muted-foreground/60 mt-0.5">
                        0x{item.id.toString(16).toUpperCase()}
                    </p>
                </div>
                <button onClick={onClose}
                    className="p-1 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all shrink-0">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>
            </div>

            <div className="flex-1 overflow-y-auto custom-scrollbar p-4 space-y-4">
                {/* Weight */}
                {(item.weight || item.weapon?.Weight || item.armor?.Weight) ? (
                    <div className="flex items-center gap-2">
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Weight</span>
                        <span className="text-[11px] font-bold text-foreground">
                            {item.weapon?.Weight ?? item.armor?.Weight ?? item.weight ?? 0}
                        </span>
                    </div>
                ) : null}

                {/* Description */}
                {item.description && (
                    <div className="space-y-1.5">
                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Description</h4>
                        <p className="text-[10px] leading-relaxed text-foreground/80 whitespace-pre-line">
                            {item.description}
                        </p>
                    </div>
                )}

                {/* Weapon Stats */}
                {item.weapon && (
                    <div className="space-y-2.5">
                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Attack Power</h4>
                        <div className="grid grid-cols-2 gap-1.5">
                            {[
                                ['Physical', item.weapon.PhysDamage],
                                ['Magic', item.weapon.MagDamage],
                                ['Fire', item.weapon.FireDamage],
                                ['Lightning', item.weapon.LitDamage],
                                ['Holy', item.weapon.HolyDamage],
                            ].filter(([, v]) => v > 0).map(([label, val]) => (
                                <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-2 py-1 border border-border/30">
                                    <span className="text-[8px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                    <span className="text-[10px] font-black text-foreground">{val}</span>
                                </div>
                            ))}
                        </div>

                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground pt-1">Scaling</h4>
                        <div className="flex gap-1.5 flex-wrap">
                            {[
                                ['STR', item.weapon.ScaleStr],
                                ['DEX', item.weapon.ScaleDex],
                                ['INT', item.weapon.ScaleInt],
                                ['FAI', item.weapon.ScaleFai],
                            ].filter(([, v]) => v > 0).map(([label, val]) => (
                                <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-2.5 py-1 border border-border/30 min-w-[40px]">
                                    <span className="text-[7px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                    <span className="text-[10px] font-black text-foreground">{val}</span>
                                </div>
                            ))}
                        </div>

                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground pt-1">Requirements</h4>
                        <div className="flex gap-1.5 flex-wrap">
                            {[
                                ['STR', item.weapon.ReqStr],
                                ['DEX', item.weapon.ReqDex],
                                ['INT', item.weapon.ReqInt],
                                ['FAI', item.weapon.ReqFai],
                                ['ARC', item.weapon.ReqArc],
                            ].filter(([, v]) => v > 0).map(([label, val]) => (
                                <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-2.5 py-1 border border-border/30 min-w-[40px]">
                                    <span className="text-[7px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                    <span className="text-[10px] font-black text-foreground">{val}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {/* Armor Stats */}
                {item.armor && (
                    <div className="space-y-2.5">
                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Damage Negation</h4>
                        <div className="grid grid-cols-2 gap-1.5">
                            {[
                                ['Physical', item.armor.Physical],
                                ['Strike', item.armor.Strike],
                                ['Slash', item.armor.Slash],
                                ['Pierce', item.armor.Pierce],
                                ['Magic', item.armor.Magic],
                                ['Fire', item.armor.Fire],
                                ['Lightning', item.armor.Lightning],
                                ['Holy', item.armor.Holy],
                            ].map(([label, val]) => (
                                <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-2 py-1 border border-border/30">
                                    <span className="text-[8px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                    <span className="text-[10px] font-black text-foreground">{(val as number).toFixed(1)}%</span>
                                </div>
                            ))}
                        </div>

                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground pt-1">Resistance</h4>
                        <div className="grid grid-cols-2 gap-1.5">
                            {[
                                ['Immunity', item.armor.Immunity],
                                ['Robustness', item.armor.Robustness],
                                ['Focus', item.armor.Focus],
                                ['Vitality', item.armor.Vitality],
                            ].map(([label, val]) => (
                                <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-2 py-1 border border-border/30">
                                    <span className="text-[8px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                    <span className="text-[10px] font-black text-foreground">{val}</span>
                                </div>
                            ))}
                        </div>

                        {item.armor.Poise > 0 && (
                            <div className="flex items-center gap-2 pt-1">
                                <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Poise</span>
                                <span className="text-[11px] font-bold text-foreground">{item.armor.Poise.toFixed(1)}</span>
                            </div>
                        )}
                    </div>
                )}

                {/* Spell Stats */}
                {item.spell && (
                    <div className="space-y-2.5">
                        <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Spell Info</h4>
                        <div className="grid grid-cols-2 gap-1.5">
                            <div className="flex items-center justify-between bg-muted/20 rounded px-2 py-1 border border-border/30">
                                <span className="text-[8px] font-bold uppercase tracking-wider text-muted-foreground">FP Cost</span>
                                <span className="text-[10px] font-black text-foreground">{item.spell.FPCost}</span>
                            </div>
                            <div className="flex items-center justify-between bg-muted/20 rounded px-2 py-1 border border-border/30">
                                <span className="text-[8px] font-bold uppercase tracking-wider text-muted-foreground">Slots</span>
                                <span className="text-[10px] font-black text-foreground">{item.spell.Slots}</span>
                            </div>
                        </div>

                        {(item.spell.ReqInt > 0 || item.spell.ReqFai > 0 || item.spell.ReqArc > 0) && (
                            <>
                                <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground pt-1">Requirements</h4>
                                <div className="flex gap-1.5 flex-wrap">
                                    {[
                                        ['INT', item.spell.ReqInt],
                                        ['FAI', item.spell.ReqFai],
                                        ['ARC', item.spell.ReqArc],
                                    ].filter(([, v]) => v > 0).map(([label, val]) => (
                                        <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-2.5 py-1 border border-border/30 min-w-[40px]">
                                            <span className="text-[7px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                            <span className="text-[10px] font-black text-foreground">{val}</span>
                                        </div>
                                    ))}
                                </div>
                            </>
                        )}
                    </div>
                )}

                {/* Item info */}
                <div className="space-y-1.5 pt-2 border-t border-border/30">
                    <h4 className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">Item Info</h4>
                    <div className="grid grid-cols-2 gap-1.5 text-[9px]">
                        <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                            <span className="text-muted-foreground font-bold">Max Inventory</span>
                            <span className="font-black text-foreground">{item.maxInventory}</span>
                        </div>
                        <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                            <span className="text-muted-foreground font-bold">Max Storage</span>
                            <span className="font-black text-foreground">{item.maxStorage}</span>
                        </div>
                        {item.maxUpgrade > 0 && (
                            <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                                <span className="text-muted-foreground font-bold">Max Upgrade</span>
                                <span className="font-black text-foreground">+{item.maxUpgrade}</span>
                            </div>
                        )}
                    </div>
                </div>

                {/* No data fallback */}
                {!item.description && !item.weapon && !item.armor && !item.spell && (
                    <p className="text-[9px] text-muted-foreground/60 italic">No description or stats available for this item.</p>
                )}
            </div>
        </div>
    );
}
