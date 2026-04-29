interface CategorySelectProps {
    value: string;
    onChange: (value: string) => void;
    className?: string;
}

// 18 categories in exact in-game order — see spec/36 and Inventory-refactor.md.
// Labels match the in-game UI 1:1 ("Ashes" not "Spirit Ashes", "Info" not
// "Information", "/" separator instead of "&").
const GAME_CATEGORIES: ReadonlyArray<{ value: string; label: string }> = [
    { value: 'tools', label: 'Tools' },
    { value: 'ashes', label: 'Ashes' },
    { value: 'crafting_materials', label: 'Crafting Materials' },
    { value: 'bolstering_materials', label: 'Bolstering Materials' },
    { value: 'key_items', label: 'Key Items' },
    { value: 'sorceries', label: 'Sorceries' },
    { value: 'incantations', label: 'Incantations' },
    { value: 'ashes_of_war', label: 'Ashes of War' },
    { value: 'melee_armaments', label: 'Melee Armaments' },
    { value: 'ranged_and_catalysts', label: 'Ranged Weapons / Catalysts' },
    { value: 'arrows_and_bolts', label: 'Arrows / Bolts' },
    { value: 'shields', label: 'Shields' },
    { value: 'head', label: 'Head' },
    { value: 'chest', label: 'Chest' },
    { value: 'arms', label: 'Arms' },
    { value: 'legs', label: 'Legs' },
    { value: 'talismans', label: 'Talismans' },
    { value: 'info', label: 'Info' },
];

export const CATEGORY_VALUES: ReadonlyArray<string> = GAME_CATEGORIES.map(c => c.value);

export function CategorySelect({ value, onChange, className }: CategorySelectProps) {
    return (
        <div className={`relative ${className ?? 'w-56'}`}>
            <select
                value={value}
                onChange={e => onChange(e.target.value)}
                className="w-full appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest text-muted-foreground outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all cursor-pointer"
            >
                <option value="all">All Categories</option>
                {GAME_CATEGORIES.map(c => (
                    <option key={c.value} value={c.value}>{c.label}</option>
                ))}
            </select>
            <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted-foreground">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M19 9l-7 7-7-7"></path></svg>
            </div>
        </div>
    );
}
