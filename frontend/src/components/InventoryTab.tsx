import {useEffect, useState} from 'react';
import {GetItemList, GetCharacter} from '../../wailsjs/go/main/App';
import {db, vm} from '../../wailsjs/go/models';

interface InventoryTabProps {
    charIndex: number;
    columnVisibility: {
        id: boolean;
        category: boolean;
    };
}

export function InventoryTab({ charIndex, columnVisibility }: InventoryTabProps) {
    const [mode, setMode] = useState<'database' | 'character' | 'storage'>('character');
    const [category, setCategory] = useState('all');
    const [search, setSearch] = useState('');
    const [dbItems, setDbItems] = useState<db.ItemEntry[]>([]);
    const [charInventory, setCharInventory] = useState<vm.ItemViewModel[]>([]);
    const [charStorage, setCharStorage] = useState<vm.ItemViewModel[]>([]);
    const [loading, setLoading] = useState(false);
    
    // Sorting state
    const [sortCol, setSortCol] = useState<string>('name');
    const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

    const [selectedIcon, setSelectedIcon] = useState<{name: string, path: string} | null>(null);

    const getItemIconPath = (name: string, category: string) => {
        let cleanName = name.toLowerCase();

        // 1. Final character normalization (only letters, numbers, and underscores)
        cleanName = cleanName
            .replace(/'/g, '')
            .replace(/\s+/g, '_')
            .replace(/-/g, '_')
            .replace(/[^\w]/g, '') // Remove everything except letters, numbers, and underscores
            .replace(/_+/g, '_')   // Collapse multiple underscores
            .replace(/^_+|_+$/g, ''); // Trim underscores from ends

        // 2. Special cases
        if (cleanName === 'golden_vow' && (category.toLowerCase() === 'ash of war' || category.toLowerCase() === 'aows' || category.toLowerCase() === 'ashes')) {
            cleanName = 'ashes_of_war_golden_vow';
        }
        
        let catDir = category.toLowerCase();
        
        // Map categories to folder names
        if (catDir === 'weapon' || catDir === 'weapons') {
            catDir = 'weapons';
        } else if (catDir === 'armor' || catDir === 'armors') {
            catDir = 'armor';
        } else if (catDir === 'item' || catDir === 'items' || catDir === 'goods') {
            catDir = 'goods';
        } else if (catDir === 'ash of war' || catDir === 'aows' || catDir === 'ashes') {
            catDir = 'ashes';
        } else if (catDir === 'talisman' || catDir === 'talismans') {
            catDir = 'talismans';
        }
        
        return `items/${catDir}/${cleanName}.png`;
    };

    const handleImageError = (e: React.SyntheticEvent<HTMLImageElement, Event>) => {
        const target = e.currentTarget;
        target.style.display = 'none';
        const parent = target.parentElement;
        if (parent) {
            const placeholder = document.createElement('div');
            placeholder.className = 'text-[10px] font-black text-muted-foreground/30 select-none';
            placeholder.innerText = '?';
            parent.appendChild(placeholder);
        }
    };

    useEffect(() => {
        setLoading(true);
        if (mode === 'database') {
            // If mode is database and category is 'all', default to 'weapons'
            const fetchCat = category === 'all' ? 'weapons' : category;
            GetItemList(fetchCat).then(res => {
                setDbItems(res || []);
                setLoading(false);
            });
        } else {
            GetCharacter(charIndex).then(res => {
                setCharInventory(res?.inventory || []);
                setCharStorage(res?.storage || []);
                setLoading(false);
            }).catch(err => {
                console.error(err);
                setLoading(false);
            });
        }
    }, [mode, category, charIndex]);

    const handleSort = (col: string) => {
        if (sortCol === col) {
            setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
        } else {
            setSortCol(col);
            setSortDir('asc');
        }
    };

    const sortItems = (items: any[]) => {
        return [...items].sort((a, b) => {
            let valA = a[sortCol as keyof typeof a];
            let valB = b[sortCol as keyof typeof b];

            if (typeof valA === 'string') {
                valA = valA.toLowerCase();
                valB = valB.toLowerCase();
            }

            if (valA < valB) return sortDir === 'asc' ? -1 : 1;
            if (valA > valB) return sortDir === 'asc' ? 1 : -1;
            return 0;
        });
    };

    const filteredDbItems = sortItems(dbItems.filter(item => 
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.id.toString(16).toLowerCase().includes(search.toLowerCase())
    ));

    const activeItems = mode === 'character' ? charInventory : charStorage;
    const filteredOwnedItems = sortItems(activeItems.filter(item => {
        const matchesSearch = item.name.toLowerCase().includes(search.toLowerCase()) ||
                            item.category.toLowerCase().includes(search.toLowerCase()) ||
                            item.id.toString(16).toLowerCase().includes(search.toLowerCase());
        
        if (category === 'all') return matchesSearch;
        
        // Map internal category names to selector values
        const itemCat = item.category.toLowerCase();
        if (category === 'weapons' && itemCat === 'weapon') return matchesSearch;
        if (category === 'armors' && itemCat === 'armor') return matchesSearch;
        if (category === 'items' && itemCat === 'item') return matchesSearch;
        if (category === 'talismans' && itemCat === 'talisman') return matchesSearch;
        if (category === 'aows' && itemCat === 'ash of war') return matchesSearch;
        
        return false;
    }));

    const SortIndicator = ({ col }: { col: string }) => {
        if (sortCol !== col) return <span className="ml-1 opacity-20">↕</span>;
        return <span className="ml-1 text-primary">{sortDir === 'asc' ? '↑' : '↓'}</span>;
    };

    return (
        <div className="flex-1 flex flex-col min-h-0 space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Icon Popover */}
            {selectedIcon && (
                <div 
                    className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300"
                    onClick={() => setSelectedIcon(null)}
                >
                    <div className="card p-8 flex flex-col items-center space-y-6 max-w-sm w-full mx-4 shadow-2xl shadow-primary/20 border-primary/20 animate-in zoom-in-95 duration-300">
                        <div className="relative group">
                            <div className="absolute -inset-4 bg-primary/10 rounded-full blur-2xl group-hover:bg-primary/20 transition-all duration-500" />
                            <img 
                                src={selectedIcon.path} 
                                alt={selectedIcon.name}
                                className="w-48 h-48 object-contain relative z-10 drop-shadow-[0_0_15px_rgba(var(--primary),0.3)]"
                                onError={(e) => (e.currentTarget.src = '/src/assets/images/logo-universal.png')}
                            />
                        </div>
                        <div className="text-center space-y-2">
                            <h3 className="text-lg font-black uppercase tracking-widest text-foreground">{selectedIcon.name}</h3>
                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.3em]">Item Preview</p>
                        </div>
                        <button className="text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors">Click anywhere to close</button>
                    </div>
                </div>
            )}

            {/* Mode Toggle & Search Bar */}
            <div className="flex flex-col md:flex-row gap-4 shrink-0">
                <div className="flex bg-muted/30 p-1 rounded-lg border border-border w-full md:w-auto">
                    <button 
                        onClick={() => setMode('character')}
                        className={`px-4 py-2 rounded-md text-[10px] font-black uppercase tracking-widest transition-all ${mode === 'character' ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        Inventory
                    </button>
                    <button 
                        onClick={() => setMode('storage')}
                        className={`px-4 py-2 rounded-md text-[10px] font-black uppercase tracking-widest transition-all ${mode === 'storage' ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        Storage
                    </button>
                    <button 
                        onClick={() => setMode('database')}
                        className={`px-4 py-2 rounded-md text-[10px] font-black uppercase tracking-widest transition-all ${mode === 'database' ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        Database
                    </button>
                </div>

                <div className="relative w-full md:w-48">
                    <select 
                        value={category}
                        onChange={e => setCategory(e.target.value)}
                        className="w-full appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest text-muted-foreground outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all cursor-pointer"
                    >
                        {mode !== 'database' && <option value="all">All Categories</option>}
                        <option value="weapons">Weapons</option>
                        <option value="armors">Armors</option>
                        <option value="items">Items</option>
                        <option value="talismans">Talismans</option>
                        <option value="aows">Ashes of War</option>
                    </select>
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted-foreground">
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M19 9l-7 7-7-7"></path></svg>
                    </div>
                </div>
                
                <div className="relative flex-1">
                    <input 
                        type="text" 
                        placeholder={mode === 'database' ? "Search database..." : "Search owned items..."}
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        className="w-full bg-muted/30 border border-border rounded-md px-10 py-2.5 text-xs font-semibold focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                    />
                    <div className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                    </div>
                </div>
            </div>

            {/* Table Card */}
            <div className="card overflow-hidden flex flex-col flex-1 min-h-0">
                <div className="overflow-y-auto flex-1 custom-scrollbar">
                    <table className="w-full text-left text-sm border-collapse">
                        <thead className="bg-muted/30 text-[10px] font-black text-muted-foreground uppercase tracking-[0.2em] sticky top-0 z-10 backdrop-blur-md border-b border-border">
                            <tr>
                                {columnVisibility.id && (
                                    <th className="px-6 py-4 cursor-pointer hover:text-foreground transition-colors" onClick={() => handleSort('id')}>
                                        ID (Hex) <SortIndicator col="id" />
                                    </th>
                                )}
                                <th className="px-6 py-4 cursor-pointer hover:text-foreground transition-colors" onClick={() => handleSort('name')}>
                                    Designation <SortIndicator col="name" />
                                </th>
                                {columnVisibility.category && (
                                    <th className="px-6 py-4 cursor-pointer hover:text-foreground transition-colors" onClick={() => handleSort('category')}>
                                        Category <SortIndicator col="category" />
                                    </th>
                                )}
                                {mode !== 'database' && (
                                    <th className="px-6 py-4 text-right cursor-pointer hover:text-foreground transition-colors" onClick={() => handleSort('quantity')}>
                                        Qty <SortIndicator col="quantity" />
                                    </th>
                                )}
                                {mode === 'database' && <th className="px-6 py-4 text-right">Action</th>}
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/30">
                            {loading ? (
                                <tr>
                                    <td colSpan={5} className="px-6 py-24 text-center">
                                        <div className="flex flex-col items-center justify-center space-y-4">
                                            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Accessing data...</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : mode !== 'database' ? (
                                filteredOwnedItems.length > 0 ? (
                                    filteredOwnedItems.map((item, idx) => (
                                        <tr key={`${item.handle}-${idx}`} className="hover:bg-muted/20 transition-colors group">
                                            {columnVisibility.id && (
                                                <td className="px-6 py-4 font-mono text-[11px] text-muted-foreground tracking-tighter">
                                                    {item.id.toString(16).toUpperCase().padStart(8, '0')}
                                                </td>
                                            )}
                                            <td className="px-6 py-4 font-bold text-foreground text-xs">
                                                <div 
                                                    className="flex items-center space-x-3 cursor-pointer group/item"
                                                    onClick={() => setSelectedIcon({ name: item.name, path: getItemIconPath(item.name, item.category) })}
                                                >
                                                    <div className="w-8 h-8 rounded bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden group-hover/item:border-primary/50 transition-all">
                                                        <img 
                                                            src={getItemIconPath(item.name, item.category)} 
                                                            alt="" 
                                                            className="w-6 h-6 object-contain opacity-80 group-hover/item:opacity-100 group-hover/item:scale-110 transition-all"
                                                            onError={handleImageError}
                                                        />
                                                    </div>
                                                    <span className={item.name.startsWith('Unknown Item') ? 'text-muted-foreground italic font-medium opacity-60' : ''}>
                                                        {item.name}
                                                    </span>
                                                </div>
                                            </td>
                                            {columnVisibility.category && (
                                                <td className="px-6 py-4">
                                                    <span className="text-[9px] font-black uppercase tracking-widest px-2 py-0.5 rounded bg-muted/50 text-muted-foreground">
                                                        {item.category}
                                                    </span>
                                                </td>
                                            )}
                                            <td className="px-6 py-4 text-right font-mono text-xs text-primary font-bold">
                                                {item.quantity}
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-24 text-center">
                                            <p className="text-xs text-muted-foreground font-medium italic">Nothing found in this section.</p>
                                        </td>
                                    </tr>
                                )
                            ) : filteredDbItems.length > 0 ? (
                                filteredDbItems.map(item => (
                                    <tr key={item.id} className="hover:bg-muted/20 transition-colors group">
                                        {columnVisibility.id && (
                                            <td className="px-6 py-4 font-mono text-[11px] text-muted-foreground tracking-tighter">
                                                {item.id.toString(16).toUpperCase().padStart(8, '0')}
                                            </td>
                                        )}
                                        <td className="px-6 py-4 font-bold text-foreground text-xs">
                                            <div 
                                                className="flex items-center space-x-3 cursor-pointer group/item"
                                                onClick={() => setSelectedIcon({ name: item.name, path: getItemIconPath(item.name, item.category) })}
                                            >
                                                <div className="w-8 h-8 rounded bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden group-hover/item:border-primary/50 transition-all">
                                                    <img 
                                                        src={getItemIconPath(item.name, item.category)} 
                                                        alt="" 
                                                        className="w-6 h-6 object-contain opacity-80 group-hover/item:opacity-100 group-hover/item:scale-110 transition-all"
                                                        onError={handleImageError}
                                                    />
                                                </div>
                                                <span className={item.name.startsWith('Unknown Item') ? 'text-muted-foreground italic font-medium opacity-60' : ''}>
                                                    {item.name}
                                                </span>
                                            </div>
                                        </td>
                                        <td colSpan={2} className="px-6 py-4 text-right">
                                            <button className="text-[9px] font-black uppercase tracking-[0.2em] text-muted-foreground hover:text-primary transition-colors px-3 py-1 border border-transparent hover:border-primary/30 rounded">
                                                Add to bag
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            ) : (
                                <tr>
                                    <td colSpan={5} className="px-6 py-24 text-center">
                                        <p className="text-xs text-muted-foreground font-medium italic">No results found in the Lands Between.</p>
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
                <div className="px-6 py-3 bg-muted/10 text-[9px] font-black text-muted-foreground uppercase tracking-[0.2em] border-t border-border flex justify-between items-center">
                    <div className="flex items-center space-x-4">
                        <span>Total: {mode === 'database' ? filteredDbItems.length : filteredOwnedItems.length}</span>
                        <span className="w-1 h-1 bg-border rounded-full" />
                        <span>Mode: {mode}</span>
                    </div>
                    <span className="opacity-50">Verified Integrity</span>
                </div>
            </div>
        </div>
    );
}
