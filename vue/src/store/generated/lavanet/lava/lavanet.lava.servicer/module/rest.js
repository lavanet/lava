/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */
export var ContentType;
(function (ContentType) {
    ContentType["Json"] = "application/json";
    ContentType["FormData"] = "multipart/form-data";
    ContentType["UrlEncoded"] = "application/x-www-form-urlencoded";
})(ContentType || (ContentType = {}));
export class HttpClient {
    constructor(apiConfig = {}) {
        this.baseUrl = "";
        this.securityData = null;
        this.securityWorker = null;
        this.abortControllers = new Map();
        this.baseApiParams = {
            credentials: "same-origin",
            headers: {},
            redirect: "follow",
            referrerPolicy: "no-referrer",
        };
        this.setSecurityData = (data) => {
            this.securityData = data;
        };
        this.contentFormatters = {
            [ContentType.Json]: (input) => input !== null && (typeof input === "object" || typeof input === "string") ? JSON.stringify(input) : input,
            [ContentType.FormData]: (input) => Object.keys(input || {}).reduce((data, key) => {
                data.append(key, input[key]);
                return data;
            }, new FormData()),
            [ContentType.UrlEncoded]: (input) => this.toQueryString(input),
        };
        this.createAbortSignal = (cancelToken) => {
            if (this.abortControllers.has(cancelToken)) {
                const abortController = this.abortControllers.get(cancelToken);
                if (abortController) {
                    return abortController.signal;
                }
                return void 0;
            }
            const abortController = new AbortController();
            this.abortControllers.set(cancelToken, abortController);
            return abortController.signal;
        };
        this.abortRequest = (cancelToken) => {
            const abortController = this.abortControllers.get(cancelToken);
            if (abortController) {
                abortController.abort();
                this.abortControllers.delete(cancelToken);
            }
        };
        this.request = ({ body, secure, path, type, query, format = "json", baseUrl, cancelToken, ...params }) => {
            const secureParams = (secure && this.securityWorker && this.securityWorker(this.securityData)) || {};
            const requestParams = this.mergeRequestParams(params, secureParams);
            const queryString = query && this.toQueryString(query);
            const payloadFormatter = this.contentFormatters[type || ContentType.Json];
            return fetch(`${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`, {
                ...requestParams,
                headers: {
                    ...(type && type !== ContentType.FormData ? { "Content-Type": type } : {}),
                    ...(requestParams.headers || {}),
                },
                signal: cancelToken ? this.createAbortSignal(cancelToken) : void 0,
                body: typeof body === "undefined" || body === null ? null : payloadFormatter(body),
            }).then(async (response) => {
                const r = response;
                r.data = null;
                r.error = null;
                const data = await response[format]()
                    .then((data) => {
                    if (r.ok) {
                        r.data = data;
                    }
                    else {
                        r.error = data;
                    }
                    return r;
                })
                    .catch((e) => {
                    r.error = e;
                    return r;
                });
                if (cancelToken) {
                    this.abortControllers.delete(cancelToken);
                }
                if (!response.ok)
                    throw data;
                return data;
            });
        };
        Object.assign(this, apiConfig);
    }
    addQueryParam(query, key) {
        const value = query[key];
        return (encodeURIComponent(key) +
            "=" +
            encodeURIComponent(Array.isArray(value) ? value.join(",") : typeof value === "number" ? value : `${value}`));
    }
    toQueryString(rawQuery) {
        const query = rawQuery || {};
        const keys = Object.keys(query).filter((key) => "undefined" !== typeof query[key]);
        return keys
            .map((key) => typeof query[key] === "object" && !Array.isArray(query[key])
            ? this.toQueryString(query[key])
            : this.addQueryParam(query, key))
            .join("&");
    }
    addQueryParams(rawQuery) {
        const queryString = this.toQueryString(rawQuery);
        return queryString ? `?${queryString}` : "";
    }
    mergeRequestParams(params1, params2) {
        return {
            ...this.baseApiParams,
            ...params1,
            ...(params2 || {}),
            headers: {
                ...(this.baseApiParams.headers || {}),
                ...(params1.headers || {}),
                ...((params2 && params2.headers) || {}),
            },
        };
    }
}
/**
 * @title servicer/block_deadline_for_callback.proto
 * @version version not set
 */
export class Api extends HttpClient {
    constructor() {
        super(...arguments);
        /**
         * No description
         *
         * @tags Query
         * @name QueryBlockDeadlineForCallback
         * @summary Queries a BlockDeadlineForCallback by index.
         * @request GET:/lavanet/lava/servicer/block_deadline_for_callback
         */
        this.queryBlockDeadlineForCallback = (params = {}) => this.request({
            path: `/lavanet/lava/servicer/block_deadline_for_callback`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryGetPairing
         * @summary Queries a list of GetPairing items.
         * @request GET:/lavanet/lava/servicer/get_pairing/{specName}/{userAddr}
         */
        this.queryGetPairing = (specName, userAddr, params = {}) => this.request({
            path: `/lavanet/lava/servicer/get_pairing/${specName}/${userAddr}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryParams
         * @summary Parameters queries the parameters of the module.
         * @request GET:/lavanet/lava/servicer/params
         */
        this.queryParams = (params = {}) => this.request({
            path: `/lavanet/lava/servicer/params`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QuerySpecStakeStorageAll
         * @summary Queries a list of SpecStakeStorage items.
         * @request GET:/lavanet/lava/servicer/spec_stake_storage
         */
        this.querySpecStakeStorageAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/servicer/spec_stake_storage`,
            method: "GET",
            query: query,
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QuerySpecStakeStorage
         * @summary Queries a SpecStakeStorage by index.
         * @request GET:/lavanet/lava/servicer/spec_stake_storage/{index}
         */
        this.querySpecStakeStorage = (index, params = {}) => this.request({
            path: `/lavanet/lava/servicer/spec_stake_storage/${index}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryStakeMapAll
         * @summary Queries a list of StakeMap items.
         * @request GET:/lavanet/lava/servicer/stake_map
         */
        this.queryStakeMapAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/servicer/stake_map`,
            method: "GET",
            query: query,
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryStakeMap
         * @summary Queries a StakeMap by index.
         * @request GET:/lavanet/lava/servicer/stake_map/{index}
         */
        this.queryStakeMap = (index, params = {}) => this.request({
            path: `/lavanet/lava/servicer/stake_map/${index}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryStakedServicers
         * @summary Queries a list of StakedServicers items.
         * @request GET:/lavanet/lava/servicer/staked_servicers/{specName}
         */
        this.queryStakedServicers = (specName, params = {}) => this.request({
            path: `/lavanet/lava/servicer/staked_servicers/${specName}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUnstakingServicersAllSpecsAll
         * @summary Queries a list of UnstakingServicersAllSpecs items.
         * @request GET:/lavanet/lava/servicer/unstaking_servicers_all_specs
         */
        this.queryUnstakingServicersAllSpecsAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/servicer/unstaking_servicers_all_specs`,
            method: "GET",
            query: query,
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUnstakingServicersAllSpecs
         * @summary Queries a UnstakingServicersAllSpecs by id.
         * @request GET:/lavanet/lava/servicer/unstaking_servicers_all_specs/{id}
         */
        this.queryUnstakingServicersAllSpecs = (id, params = {}) => this.request({
            path: `/lavanet/lava/servicer/unstaking_servicers_all_specs/${id}`,
            method: "GET",
            format: "json",
            ...params,
        });
    }
}
