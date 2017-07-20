/*
  This file is part of rifthash.

  rifthash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  rifthash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with rifthash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file rifthash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define RIFTHASH_REVISION 23
#define RIFTHASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define RIFTHASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define RIFTHASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define RIFTHASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define RIFTHASH_EPOCH_LENGTH 30000U
#define RIFTHASH_MIX_BYTES 128
#define RIFTHASH_HASH_BYTES 64
#define RIFTHASH_DATASET_PARENTS 256
#define RIFTHASH_CACHE_ROUNDS 3
#define RIFTHASH_ACCESSES 64
#define RIFTHASH_DAG_MAGIC_NUM_SIZE 8
#define RIFTHASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct rifthash_h256 { uint8_t b[32]; } rifthash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// rifthash_h256_t a = rifthash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define rifthash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct rifthash_light;
typedef struct rifthash_light* rifthash_light_t;
struct rifthash_full;
typedef struct rifthash_full* rifthash_full_t;
typedef int(*rifthash_callback_t)(unsigned);

typedef struct rifthash_return_value {
	rifthash_h256_t result;
	rifthash_h256_t mix_hash;
	bool success;
} rifthash_return_value_t;

/**
 * Allocate and initialize a new rifthash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated rifthash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref rifthash_compute_cache_nodes()
 */
rifthash_light_t rifthash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated rifthash_light handler
 * @param light        The light handler to free
 */
void rifthash_light_delete(rifthash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of rifthash_return_value_t holding the return values
 */
rifthash_return_value_t rifthash_light_compute(
	rifthash_light_t light,
	rifthash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new rifthash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref rifthash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated rifthash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref rifthash_compute_full_data()
 */
rifthash_full_t rifthash_full_new(rifthash_light_t light, rifthash_callback_t callback);

/**
 * Frees a previously allocated rifthash_full handler
 * @param full    The light handler to free
 */
void rifthash_full_delete(rifthash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of rifthash_return_value to hold the return value
 */
rifthash_return_value_t rifthash_full_compute(
	rifthash_full_t full,
	rifthash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* rifthash_full_dag(rifthash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t rifthash_full_dag_size(rifthash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
rifthash_h256_t rifthash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
