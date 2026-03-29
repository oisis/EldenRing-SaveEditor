import {useEffect, useState} from 'react';
import {GetItemList} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

export function InventoryTab() {
    const [category, setCategory] = useState('weapons');
    const [items, setItems] = useState<db.ItemEntry[]>([]);
    const [search, setSearch] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setLoading(true);
        GetItemList(category)
            .then(res => {
                setItems(res || []);
                setLoading(false);
            })
            .catch(err => {
                console.error(err);
                setLoading(false);
            });
    }, [category]);

    const filteredItems = items.filter(item => 
        item.name.toLowerCase().includes(search.toLowerCase()) ||
        item.id.toString(16).includes(search.toLowerCase())
    );

    return (
        <div className="space-y-6 animate-in fade-in duration-500">
            {/* Search & Filter */}
            <div className="flex flex-col sm:flex-row gap-3">
                <div className="relative w-full sm:w-48">
                    <select 
                        value={category} 
                        onChange={e => setCategory(e.target.value)}
                        className="w-full appearance-none bg-muted/50 border border-border rounded px-3 py-2 pr-10 text-xs font-semibold text-muted-foreground uppercase tracking-wider outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all cursor-pointer"
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
                        className="w-full bg-muted/50 border border-border rounded px-10 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                    />
                    <div className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                    </div>
                </div>
            </div>

            {/* Table */}
            <div className="border border-border rounded-lg overflow-hidden flex flex-col h-[550px] bg-background">
                <div className="overflow-y-auto flex-1 custom-scrollbar">
                    <table className="w-full text-left text-sm border-collapse">
                        <thead className="bg-muted/50 text-xs font-semibold text-muted-foreground uppercase tracking-wider sticky top-0 z-10 backdrop-blur-md border-b border-border">
                            <tr>
                                <th className="px-6 py-3 font-semibold">ID</th>
                                <th className="px-6 py-3 font-semibold">Designation</th>
                                <th className="px-6 py-3 font-semibold text-right">Action</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/40">
                            {loading ? (
                                <tr>
                                    <td colSpan={3} className="px-6 py-24 text-center">
                                        <div className="flex flex-col items-center justify-center space-y-3">
                                            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                                            <p className="text-xs font-medium text-muted-foreground">Querying database...</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : filteredItems.length > 0 ? (
                                filteredItems.map(item => (
                                    <tr key={item.id} className="hover:bg-muted/30 transition-colors group">
                                        <td className="px-6 py-3 font-mono text-xs text-muted-foreground">
                                            {item.id.toString(16).toUpperCase().padStart(8, '0')}
                                        </td>
                                        <td className="px-6 py-3 font-medium text-foreground">
                                            {item.name}
                                        </td>
                                        <td className="px-6 py-3 text-right">
                                            <button className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground hover:text-blue-500 transition-colors px-3 py-1">
                                                Add to bag
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            ) : (
                                <tr>
                                    <td colSpan={3} className="px-6 py-24 text-center">
                                        <p className="text-sm text-muted-foreground italic">No results found.</p>
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
                <div className="px-6 py-3 bg-muted/20 text-[10px] font-bold text-muted-foreground uppercase tracking-widest border-t border-border flex justify-between items-center">
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
