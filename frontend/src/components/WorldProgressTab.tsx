import {useEffect, useState} from 'react';
import {GetAllGraces} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

export function WorldProgressTab() {
    const [graces, setGraces] = useState<db.GraceEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [expandedRegions, setExpandedRegions] = useState<{[key: string]: boolean}>({});

    useEffect(() => {
        setLoading(true);
        GetAllGraces()
            .then(res => {
                setGraces(res || []);
                // Expand first region by default
                if (res && res.length > 0) {
                    setExpandedRegions({[res[0].region]: true});
                }
                setLoading(false);
            })
            .catch(err => {
                console.error(err);
                setLoading(false);
            });
    }, []);

    // Group graces by region
    const regions = graces.reduce((acc, grace) => {
        if (!acc[grace.region]) acc[grace.region] = [];
        acc[grace.region].push(grace);
        return acc;
    }, {} as {[key: string]: db.GraceEntry[]});

    const toggleRegion = (region: string) => {
        setExpandedRegions(prev => ({...prev, [region]: !prev[region]}));
    };

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-xs font-medium text-muted-foreground">Scanning world data...</p>
        </div>
    );

    return (
        <div className="space-y-8 animate-in fade-in duration-500 pb-12">
            <div className="flex items-center space-x-2">
                <div className="w-1 h-4 bg-blue-500 rounded-full" />
                <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Sites of Grace</h3>
            </div>
            
            <div className="space-y-2">
                {Object.entries(regions).sort().map(([region, regionGraces]) => (
                    <div key={region} className="border border-border rounded-lg overflow-hidden bg-background">
                        <button 
                            onClick={() => toggleRegion(region)}
                            className={`w-full px-5 py-3 flex justify-between items-center transition-colors ${expandedRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/20'}`}
                        >
                            <div className="flex items-center space-x-3">
                                <div className={`transition-transform duration-200 ${expandedRegions[region] ? 'rotate-90 text-blue-500' : 'text-muted-foreground'}`}>
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7"></path>
                                    </svg>
                                </div>
                                <h2 className="text-sm font-semibold text-foreground">{region}</h2>
                            </div>
                            <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest bg-muted/50 px-2 py-0.5 rounded">
                                {regionGraces.length}
                            </span>
                        </button>
                        
                        {expandedRegions[region] && (
                            <div className="p-5 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-2.5 animate-in slide-in-from-top-1 duration-200">
                                {regionGraces.map(grace => (
                                    <label key={grace.id} className="flex items-center space-x-3 group cursor-pointer py-1 px-2 rounded hover:bg-muted/40 transition-colors">
                                        <div className="relative flex items-center justify-center">
                                            <input 
                                                type="checkbox" 
                                                className="peer appearance-none w-3.5 h-3.5 rounded border border-zinc-300 dark:border-zinc-700 bg-background checked:bg-blue-600 checked:border-blue-600 transition-all cursor-pointer focus:ring-2 focus:ring-blue-500/20"
                                            />
                                            <svg className="absolute w-2.5 h-2.5 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M5 13l4 4L19 7"></path>
                                            </svg>
                                        </div>
                                        <span className="text-xs text-muted-foreground group-hover:text-foreground transition-colors truncate font-medium" title={grace.name}>
                                            {grace.name}
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
