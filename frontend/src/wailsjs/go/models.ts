export namespace bridge {
	export class ConnectionInfoDTO {
	    connection_id: string;
	    hostname: string;
	    tailscale_ip: string;
	    tailscale_ipv6: string;

	    static createFrom(source: any = {}) {
	        return new ConnectionInfoDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connection_id = source["connection_id"] || "";
	        this.hostname = source["hostname"] || "";
	        this.tailscale_ip = source["tailscale_ip"] || "";
	        this.tailscale_ipv6 = source["tailscale_ipv6"] || "";
	    }
	}

	export class ForwardingDTO {
	    id: string;
	    name: string;
	    remote_port: number;
	    local_port: number;
	    enabled: boolean;
	    is_default: boolean;
	    running: boolean;
	    last_error?: string;

	    static createFrom(source: any = {}) {
	        return new ForwardingDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"] || "";
	        this.name = source["name"] || "";
	        this.remote_port = source["remote_port"] || 0;
	        this.local_port = source["local_port"] || 0;
	        this.enabled = source["enabled"] || false;
	        this.is_default = source["is_default"] || false;
	        this.running = source["running"] || false;
	        this.last_error = source["last_error"];
	    }
	}
}
