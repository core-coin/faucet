export function getAddress(address) {
    let result = null;
    let error;

    if (typeof(address) !== "string") {
        error = new Error();
        error.reason = "Address is not a string";
        throw error;
    }

    if (address.match(/^(0x)?[0-9a-fA-F]{44}$/)) {
        
        if (address.substring(0, 2) === "0x") { address = address.substring(2, address.length) }

        let prefix = address.substring(0, 2).toLowerCase();
        let checksum = address.substring(2, 4).toLowerCase();
        let pureAddress = address.substring(4, address.length).toLowerCase();
        let final = pureAddress + prefix + "00";
        let sum = "";

        if (prefix !== "ab") { 
            error = new Error();
            error.reason = "Invalid Network ID";
            throw error;
        };

        for (let i = 0; i < final.length; i++) {
            let t = final.charCodeAt(i);
            if (t >= 48 && t <= 57) {
                sum += final.charAt(i);
            } else if (t >= 97 && t <= 102) {
                sum += (t - 87);
            } else {
                error = new Error();
                error.reason = "Invalid Address";
                throw error;    
            }
        }
        let b = BigInt(sum);
        b = (98 - Number(b % 97n)).toString();
        if (b.length == 1) {b = "0" + b}
        if (b.toString() !== checksum) {
            error = new Error();
            error.reason = "Invalid Checksum";
            throw error; 
        }
    } else {
        error = new Error();
        error.reason = "Invalid address";
        throw error;
    }

    return address.toLowerCase();
}