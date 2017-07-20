#pragma once
#include "compiler.h"
#include "endian.h"
#include "rifthash.h"
#include <stdio.h>

#define ENABLE_SSE 0

#if defined(_M_X64) && ENABLE_SSE
#include <smmintrin.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

// compile time settings
#define NODE_WORDS (64/4)
#define MIX_WORDS (RIFTHASH_MIX_BYTES/4)
#define MIX_NODES (MIX_WORDS / NODE_WORDS)
#include <stdint.h>

typedef union node {
	uint8_t bytes[NODE_WORDS * 4];
	uint32_t words[NODE_WORDS];
	uint64_t double_words[NODE_WORDS / 2];

#if defined(_M_X64) && ENABLE_SSE
	__m128i xmm[NODE_WORDS/4];
#endif

} node;

static inline uint8_t rifthash_h256_get(rifthash_h256_t const* hash, unsigned int i)
{
	return hash->b[i];
}

static inline void rifthash_h256_set(rifthash_h256_t* hash, unsigned int i, uint8_t v)
{
	hash->b[i] = v;
}

static inline void rifthash_h256_reset(rifthash_h256_t* hash)
{
	memset(hash, 0, 32);
}

// Returns if hash is less than or equal to boundary (2^256/difficulty)
static inline bool rifthash_check_difficulty(
	rifthash_h256_t const* hash,
	rifthash_h256_t const* boundary
)
{
	// Boundary is big endian
	for (int i = 0; i < 32; i++) {
		if (rifthash_h256_get(hash, i) == rifthash_h256_get(boundary, i)) {
			continue;
		}
		return rifthash_h256_get(hash, i) < rifthash_h256_get(boundary, i);
	}
	return true;
}

/**
 *  Difficulty quick check for POW preverification
 *
 * @param header_hash      The hash of the header
 * @param nonce            The block's nonce
 * @param mix_hash         The mix digest hash
 * @param boundary         The boundary is defined as (2^256 / difficulty)
 * @return                 true for succesful pre-verification and false otherwise
 */
bool rifthash_quick_check_difficulty(
	rifthash_h256_t const* header_hash,
	uint64_t const nonce,
	rifthash_h256_t const* mix_hash,
	rifthash_h256_t const* boundary
);

struct rifthash_light {
	void* cache;
	uint64_t cache_size;
	uint64_t block_number;
};

/**
 * Allocate and initialize a new rifthash_light handler. Internal version
 *
 * @param cache_size    The size of the cache in bytes
 * @param seed          Block seedhash to be used during the computation of the
 *                      cache nodes
 * @return              Newly allocated rifthash_light handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref rifthash_compute_cache_nodes()
 */
rifthash_light_t rifthash_light_new_internal(uint64_t cache_size, rifthash_h256_t const* seed);

/**
 * Calculate the light client data. Internal version.
 *
 * @param light          The light client handler
 * @param full_size      The size of the full data in bytes.
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               The resulting hash.
 */
rifthash_return_value_t rifthash_light_compute_internal(
	rifthash_light_t light,
	uint64_t full_size,
	rifthash_h256_t const header_hash,
	uint64_t nonce
);

struct rifthash_full {
	FILE* file;
	uint64_t file_size;
	node* data;
};

/**
 * Allocate and initialize a new rifthash_full handler. Internal version.
 *
 * @param dirname        The directory in which to put the DAG file.
 * @param seedhash       The seed hash of the block. Used in the DAG file naming.
 * @param full_size      The size of the full data in bytes.
 * @param cache          A cache object to use that was allocated with @ref rifthash_cache_new().
 *                       Iff this function succeeds the rifthash_full_t will take memory
 *                       memory ownership of the cache and free it at deletion. If
 *                       not then the user still has to handle freeing of the cache himself.
 * @param callback       A callback function with signature of @ref rifthash_callback_t
 *                       It accepts an unsigned with which a progress of DAG calculation
 *                       can be displayed. If all goes well the callback should return 0.
 *                       If a non-zero value is returned then DAG generation will stop.
 * @return               Newly allocated rifthash_full handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref rifthash_compute_full_data()
 */
rifthash_full_t rifthash_full_new_internal(
	char const* dirname,
	rifthash_h256_t const seed_hash,
	uint64_t full_size,
	rifthash_light_t const light,
	rifthash_callback_t callback
);

void rifthash_calculate_dag_item(
	node* const ret,
	uint32_t node_index,
	rifthash_light_t const cache
);

void rifthash_quick_hash(
	rifthash_h256_t* return_hash,
	rifthash_h256_t const* header_hash,
	const uint64_t nonce,
	rifthash_h256_t const* mix_hash
);

uint64_t rifthash_get_datasize(uint64_t const block_number);
uint64_t rifthash_get_cachesize(uint64_t const block_number);

/**
 * Compute the memory data for a full node's memory
 *
 * @param mem         A pointer to an rifthash full's memory
 * @param full_size   The size of the full data in bytes
 * @param cache       A cache object to use in the calculation
 * @param callback    The callback function. Check @ref rifthash_full_new() for details.
 * @return            true if all went fine and false for invalid parameters
 */
bool rifthash_compute_full_data(
	void* mem,
	uint64_t full_size,
	rifthash_light_t const light,
	rifthash_callback_t callback
);

#ifdef __cplusplus
}
#endif
