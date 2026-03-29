import {useEffect, useState} from 'react';
import {GetItemList} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

export function InventoryTab() {
    const [category, setCategory] = useState('weapons');
    const [search, setSearch] = useState('');
    const [items, setItems] = useState<db.ItemEntry[]>([]);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setLoading(true);
        GetItemList(category).then(res => {
            setItems(res || []);
            setLoading(false);
        });
    }, [category]);

    const filteredItems = items.filter(item => 
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.id.toString(16).toLowerCase().includes(search.toLowerCase())
    );

    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Search & Filter Bar */}
            <div className="flex flex-col md:flex-row gap-4">
                <div className="relative w-full md:w-48">
                    <select 
                        value={category}
                        onChange={e => setCategory(e.target.value)}
                        className="w-full appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest text-muted-foreground outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all cursor-pointer"
                    >
                        <option value="weapons">Weapons</option>
                        <option value="armors">Armors</option>
                        <option value="items">Items</option>
                        <option value="talismans">Talismans</option>
                    </select>
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted-foreground">
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M19 9l-7 7-7-7"></path></svg>
                    </div>
                </div>
                
                <div className="relative flex-1">
                    <input 
                        type="text" 
                        placeholder="Search by name or hex ID..." 
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
            <div className="card overflow-hidden flex flex-col h-[550px]">
                <div className="overflow-y-auto flex-1 custom-scrollbar">
                    <table className="w-full text-left text-sm border-collapse">
                        <thead className="bg-muted/30 text-[10px] font-black text-muted-foreground uppercase tracking-[0.2em] sticky top-0 z-10 backdrop-blur-md border-b border-border">
                            <tr>
                                <th className="px-6 py-4">ID (Hex)</th>
                                <th className="px-6 py-4">Designation</th>
                                <th className="px-6 py-4 text-right">Action</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/30">
                            {loading ? (
                                <tr>
                                    <td colSpan={3} className="px-6 py-24 text-center">
                                        <div className="flex flex-col items-center justify-center space-y-4">
                                            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Querying database...</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : filteredItems.length > 0 ? (
                                filteredItems.map(item => (
                                    <tr key={item.id} className="hover:bg-muted/20 transition-colors group">
                                        <td className="px-6 py-4 font-mono text-[11px] text-muted-foreground tracking-tighter">
                                            {item.id.toString(16).toUpperCase().padStart(8, '0')}
                                        </td>
                                        <td className="px-6 py-4 font-bold text-foreground text-xs">
                                            {item.name}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <button className="text-[9px] font-black uppercase tracking-[0.2em] text-muted-foreground hover:text-primary transition-colors px-3 py-1 border border-transparent hover:border-primary/30 rounded">
                                                Add to bag
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            ) : (
                                <tr>
                                    <td colSpan={3} className="px-6 py-24 text-center">
                                        <p className="text-xs text-muted-foreground font-medium italic">No results found in the Lands Between.</p>
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
                <div className="px-6 py-3 bg-muted/10 text-[9px] font-black text-muted-foreground uppercase tracking-[0.2em] border-t border-border flex justify-between items-center">
                    <div className="flex items-center space-x-4">
                        <span>Total: {filteredItems.length}</span>
                        <span className="w-1 h-1 bg-border rounded-full" />
                        <span>Category: {category}</span>
                    </div>
                    <span className="opacity-50">Verified Integrity</span>
                </div>
            </div>
        </div>
    );
}
