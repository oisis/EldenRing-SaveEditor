interface CategorySelectProps {
    value: string;
    onChange: (value: string) => void;
    className?: string;
}

export function CategorySelect({ value, onChange, className }: CategorySelectProps) {
    return (
        <div className={`relative ${className ?? 'w-56'}`}>
            <select
                value={value}
                onChange={e => onChange(e.target.value)}
                className="w-full appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest text-muted-foreground outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all cursor-pointer"
            >
                <option value="all">All Categories</option>
                <optgroup label="Armaments" className="bg-background text-foreground">
                    <option value="melee_armaments">Melee Armaments</option>
                    <option value="ranged_and_catalysts">Ranged Weapons &amp; Catalysts</option>
                    <option value="arrows_and_bolts">Arrows &amp; Bolts</option>
                    <option value="shields">Shields</option>
                    <option value="ashes_of_war">Ashes of War</option>
                </optgroup>
                <optgroup label="Armor" className="bg-background text-foreground">
                    <option value="head">Head</option>
                    <option value="chest">Chest</option>
                    <option value="arms">Arms</option>
                    <option value="legs">Legs</option>
                </optgroup>
                <optgroup label="Accessories" className="bg-background text-foreground">
                    <option value="talismans">Talismans</option>
                </optgroup>
                <optgroup label="Magic" className="bg-background text-foreground">
                    <option value="sorceries">Sorceries</option>
                    <option value="incantations">Incantations</option>
                </optgroup>
                <optgroup label="Items" className="bg-background text-foreground">
                    <option value="ashes">Spirit Ashes</option>
                    <option value="tools">Tools</option>
                    <option value="crafting_materials">Crafting Materials</option>
                    <option value="bolstering_materials">Bolstering Materials</option>
                    <option value="key_items">Key Items</option>
                    <option value="info">Information</option>
                </optgroup>
            </select>
            <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted-foreground">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M19 9l-7 7-7-7"></path></svg>
            </div>
        </div>
    );
}
