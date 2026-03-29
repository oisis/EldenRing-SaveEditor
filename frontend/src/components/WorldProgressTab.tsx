import {useEffect, useState} from 'react';
import {GetAllGraces} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

export function WorldProgressTab() {
    const [graces, setGraces] = useState<db.GraceEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});

    useEffect(() => {
        setLoading(true);
        GetAllGraces().then(res => {
            setGraces(res || []);
            setLoading(false);
        });
    }, []);

    const regions = graces.reduce((acc, grace) => {
        const region = grace.name.split('(')[1]?.replace(')', '') || 'Unknown';
        if (!acc[region]) acc[region] = [];
        acc[region].push(grace);
        return acc;
    }, {} as Record<string, db.GraceEntry[]>);

    const toggleRegion = (region: string) => {
        setExpandedRegions(prev => ({...prev, [region]: !prev[region]}));
    };

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Scanning world data...</p>
        </div>
    );

    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-12">
            <div className="flex items-center space-x-2 px-1">
                <div className="w-1 h-3 bg-blue-500 rounded-full" />
                <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Sites of Grace</h3>
            </div>
            
            <div className="grid grid-cols-1 gap-4">
                {Object.entries(regions).sort().map(([region, regionGraces]) => (
                    <div key={region} className="card overflow-hidden">
                        <button 
                            onClick={() => toggleRegion(region)}
                            className={`w-full px-5 py-4 flex justify-between items-center transition-all ${expandedRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}
                        >
                            <div className="flex items-center space-x-4">
                                <div className={`transition-transform duration-300 ${expandedRegions[region] ? 'rotate-90 text-blue-500' : 'text-muted-foreground'}`}>
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7"></path>
                                    </svg>
                                </div>
                                <h2 className="text-xs font-black uppercase tracking-widest text-foreground">{region}</h2>
                            </div>
                            <span className="text-[9px] font-black text-muted-foreground uppercase tracking-widest bg-muted/50 px-2 py-0.5 rounded border border-border">
                                {regionGraces.length}
                            </span>
                        </button>
                        
                        {expandedRegions[region] && (
                            <div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-3 animate-in slide-in-from-top-2 duration-300">
                                {regionGraces.map(grace => (
                                    <label key={grace.id} className="flex items-center space-x-3 group cursor-pointer py-1.5 px-2 rounded-md hover:bg-muted/40 transition-all">
                                        <div className="relative flex items-center justify-center">
                                            <input 
                                                type="checkbox" 
                                                className="peer appearance-none w-4 h-4 rounded border border-border bg-background checked:bg-blue-600 checked:border-blue-600 transition-all cursor-pointer focus:ring-2 focus:ring-blue-500/20"
                                            />
                                            <svg className="absolute w-2.5 h-2.5 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M5 13l4 4L19 7"></path>
                                            </svg>
                                        </div>
                                        <span className="text-[11px] text-muted-foreground group-hover:text-foreground transition-colors truncate font-semibold" title={grace.name}>
                                            {grace.name.split('(')[0].trim()}
                                        </span>
                                    </label>
                                ))}
                            </div>
                        )}
                    </div>
                ))}
            </div>
        </div>
    );
}
