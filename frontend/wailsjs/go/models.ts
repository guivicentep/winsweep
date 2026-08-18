export namespace main {
	
	export class TweakDTO {
	    id: string;
	    name: string;
	    category: string;
	    description: string;
	    impact: string;
	
	    static createFrom(source: any = {}) {
	        return new TweakDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.impact = source["impact"];
	    }
	}

}

