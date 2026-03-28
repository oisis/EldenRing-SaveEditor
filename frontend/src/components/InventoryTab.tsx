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
            {/* Filter & Search Bar */}
            <div className="flex flex-col md:flex-row space-y-4 md:space-y-0 md:space-x-4 items-stretch md:items-center bg-er-gray p-4 rounded-lg border border-gray-700 shadow-lg">
                <div className="relative">
                    <select 
                        value={category} 
                        onChange={e => setCategory(e.target.value)}
                        className="appearance-none bg-er-dark border border-gray-600 rounded px-4 py-2.5 pr-10 text-sm text-er-gold outline-none focus:border-er-gold transition-all cursor-pointer"
                    >
                        <option value="weapons">Weapons</option>
                        <option value="armors">Armors</option>
                        <option value="items">Items</option>
                        <option value="talismans">Talismans</option>
                    </select>
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-er-gold/50">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path></svg>
                    </div>
                </div>
                
                <div className="flex-1 relative">
                    <input 
                        type="text" 
                        placeholder="Search items by name or ID (hex)..." 
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        className="w-full bg-er-dark border border-gray-600 rounded px-10 py-2.5 text-sm outline-none focus:border-er-gold transition-all"
                    />
                    <div className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                    </div>
                </div>
            </div>

            {/* Items Table */}
            <div className="bg-er-gray rounded-lg border border-gray-700 shadow-xl overflow-hidden">
                <div className="max-h-[600px] overflow-y-auto custom-scrollbar">
                    <table className="w-full text-left text-sm">
                        <thead className="bg-black/40 text-gray-500 uppercase text-[10px] font-bold tracking-widest sticky top-0 z-10 backdrop-blur-sm">
                            <tr>
                                <th className="px-8 py-4 border-b border-gray-800">ID (Hex)</th>
                                <th className="px-8 py-4 border-b border-gray-800">Name</th>
                                <th className="px-8 py-4 border-b border-gray-800 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-800/50">
                            {loading ? (
                                <tr>
                                    <td colSpan={3} className="px-8 py-12 text-center text-er-gold italic animate-pulse">
                                        Searching through the Lands Between...
                                    </td>
                                </tr>
                            ) : filteredItems.length > 0 ? (
                                filteredItems.map(item => (
                                    <tr key={item.id} className="hover:bg-er-gold/5 transition-colors group">
                                        <td className="px-8 py-4 font-mono text-xs text-gray-500">
                                            0x{item.id.toString(16).toUpperCase().padStart(8, '0')}
                                        </td>
                                        <td className="px-8 py-4 text-gray-300 group-hover:text-er-gold transition-colors font-medium">
                                            {item.name}
                                        </td>
                                        <td className="px-8 py-4 text-right">
                                            <button className="bg-er-gold/10 hover:bg-er-gold text-er-gold hover:text-er-dark border border-er-gold/30 px-4 py-1 rounded text-[10px] font-bold uppercase tracking-tighter transition-all">
                                                Add to Inventory
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            ) : (
                                <tr>
                                    <td colSpan={3} className="px-8 py-12 text-center text-gray-600 italic">
                                        No items found matching your search.
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
                <div className="p-4 bg-black/20 text-[10px] text-gray-600 font-bold uppercase tracking-widest flex justify-between">
                    <span>Showing {filteredItems.length} items</span>
                    <span>Database: Rust Parity v0.2.0</span>
                </div>
            </div>
        </div>
    );
}
