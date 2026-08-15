import { CloverClient, ClientOptions } from "../index.js";

/** Structural subset of Nest's DynamicModule, kept framework-free at runtime. */
export interface CloverNestDynamicModule {
  module: typeof CloverNestModule;
  providers: readonly CloverNestProvider[];
  exports: readonly [typeof CLOVER_CLIENT];
}

export interface CloverNestProvider {
  provide: typeof CLOVER_CLIENT;
  useFactory: () => CloverClient;
}

export const CLOVER_CLIENT = Symbol.for("CLOVER_CLIENT");

export interface CloverNestModuleOptions extends ClientOptions {}

/**
 * Optional NestJS adapter. Install @nestjs/common in the application and use
 * the returned structural DynamicModule from `CloverNestModule.forRoot`.
 * Keeping the implementation structural means the core SDK remains
 * tree-shakeable and usable without NestJS.
 */
export class CloverNestModule {
  static forRoot(options: CloverNestModuleOptions): CloverNestDynamicModule {
    const provider: CloverNestProvider = {
      provide: CLOVER_CLIENT,
      useFactory: () => new CloverClient(options),
    };
    return { module: CloverNestModule, providers: [provider], exports: [CLOVER_CLIENT] };
  }
}
