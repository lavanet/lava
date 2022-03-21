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
 * @title user/block_deadline_for_callback.proto
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
         * @request GET:/lavanet/lava/user/block_deadline_for_callback
         */
        this.queryBlockDeadlineForCallback = (params = {}) => this.request({
            path: `/lavanet/lava/user/block_deadline_for_callback`,
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
         * @request GET:/lavanet/lava/user/params
         */
        this.queryParams = (params = {}) => this.request({
            path: `/lavanet/lava/user/params`,
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
         * @request GET:/lavanet/lava/user/spec_stake_storage
         */
        this.querySpecStakeStorageAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/user/spec_stake_storage`,
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
         * @request GET:/lavanet/lava/user/spec_stake_storage/{index}
         */
        this.querySpecStakeStorage = (index, params = {}) => this.request({
            path: `/lavanet/lava/user/spec_stake_storage/${index}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryStakedUsers
         * @summary Queries a list of StakedUsers items.
         * @request GET:/lavanet/lava/user/staked_users/{specName}/{output}
         */
        this.queryStakedUsers = (specName, output, params = {}) => this.request({
            path: `/lavanet/lava/user/staked_users/${specName}/${output}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUnstakingUsersAllSpecsAll
         * @summary Queries a list of UnstakingUsersAllSpecs items.
         * @request GET:/lavanet/lava/user/unstaking_users_all_specs
         */
        this.queryUnstakingUsersAllSpecsAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/user/unstaking_users_all_specs`,
            method: "GET",
            query: query,
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUnstakingUsersAllSpecs
         * @summary Queries a UnstakingUsersAllSpecs by id.
         * @request GET:/lavanet/lava/user/unstaking_users_all_specs/{id}
         */
        this.queryUnstakingUsersAllSpecs = (id, params = {}) => this.request({
            path: `/lavanet/lava/user/unstaking_users_all_specs/${id}`,
            method: "GET",
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUserStakeAll
         * @summary Queries a list of UserStake items.
         * @request GET:/lavanet/lava/user/user_stake
         */
        this.queryUserStakeAll = (query, params = {}) => this.request({
            path: `/lavanet/lava/user/user_stake`,
            method: "GET",
            query: query,
            format: "json",
            ...params,
        });
        /**
         * No description
         *
         * @tags Query
         * @name QueryUserStake
         * @summary Queries a UserStake by index.
         * @request GET:/lavanet/lava/user/user_stake/{index}
         */
        this.queryUserStake = (index, params = {}) => this.request({
            path: `/lavanet/lava/user/user_stake/${index}`,
            method: "GET",
            format: "json",
            ...params,
        });
    }
}
