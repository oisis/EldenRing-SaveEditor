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

    if (loading) return <div className="text-er-gold italic animate-pulse">Scanning the Lands Between...</div>;

    return (
        <div className="space-y-6 animate-in fade-in duration-500 pb-12">
            {Object.entries(regions).sort().map(([region, regionGraces]) => (
                <div key={region} className="bg-er-gray rounded-lg border border-gray-700 shadow-lg overflow-hidden transition-all">
                    <button 
                        onClick={() => toggleRegion(region)}
                        className="w-full bg-black/30 px-6 py-4 border-b border-gray-700 flex justify-between items-center hover:bg-black/40 transition-colors"
                    >
                        <div className="flex items-center space-x-4">
                            <svg 
                                className={`w-4 h-4 text-er-gold transition-transform duration-300 ${expandedRegions[region] ? 'rotate-180' : ''}`} 
                                fill="none" stroke="currentColor" viewBox="0 0 24 24"
                            >
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7"></path>
                            </svg>
                            <h2 className="text-er-gold font-serif text-xl tracking-tight">{region}</h2>
                        </div>
                        <div className="flex items-center space-x-4">
                            <span className="text-[10px] text-gray-500 uppercase font-bold tracking-[0.2em]">{regionGraces.length} Sites of Grace</span>
                        </div>
                    </button>
                    
                    {expandedRegions[region] && (
                        <div className="p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-12 gap-y-4 animate-in slide-in-from-top-2 duration-300">
                            {regionGraces.map(grace => (
                                <label key={grace.id} className="flex items-center space-x-4 group cursor-pointer py-1">
                                    <div className="relative flex items-center justify-center">
                                        <input 
                                            type="checkbox" 
                                            className="peer appearance-none w-5 h-5 rounded border border-gray-600 bg-er-dark checked:bg-er-gold checked:border-er-gold transition-all cursor-pointer"
                                        />
                                        <svg className="absolute w-3 h-3 text-er-dark pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M5 13l4 4L19 7"></path>
                                        </svg>
                                    </div>
                                    <span className="text-sm text-gray-400 group-hover:text-er-gold transition-colors truncate" title={grace.name}>
                                        {grace.name}
                                    </span>
                                </label>
                            ))}
                        </div>
                    )}
                </div>
            ))}
        </div>
    );
}
