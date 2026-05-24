#include <cstdint>
#include <cstring>
#include <cstddef>

// ─── SHA-256 (FIPS 180-4) ────────────────────────────────────────────────────

namespace {

static const uint32_t SHA256_K[64] = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
};

static inline uint32_t rotr32(uint32_t x, int n) {
    return (x >> n) | (x << (32 - n));
}

struct SHA256Ctx {
    uint32_t state[8];
    uint64_t count;
    uint8_t  buf[64];
    uint32_t buf_len;
};

static void sha256_init(SHA256Ctx *ctx) {
    ctx->state[0] = 0x6a09e667u;
    ctx->state[1] = 0xbb67ae85u;
    ctx->state[2] = 0x3c6ef372u;
    ctx->state[3] = 0xa54ff53au;
    ctx->state[4] = 0x510e527fu;
    ctx->state[5] = 0x9b05688cu;
    ctx->state[6] = 0x1f83d9abu;
    ctx->state[7] = 0x5be0cd19u;
    ctx->count   = 0;
    ctx->buf_len = 0;
}

static void sha256_transform(SHA256Ctx *ctx, const uint8_t *block) {
    uint32_t w[64];
    for (int i = 0; i < 16; i++) {
        w[i] = ((uint32_t)block[i*4+0] << 24) |
               ((uint32_t)block[i*4+1] << 16) |
               ((uint32_t)block[i*4+2] <<  8) |
               ((uint32_t)block[i*4+3]);
    }
    for (int i = 16; i < 64; i++) {
        uint32_t s0 = rotr32(w[i-15], 7) ^ rotr32(w[i-15], 18) ^ (w[i-15] >> 3);
        uint32_t s1 = rotr32(w[i-2], 17) ^ rotr32(w[i-2],  19) ^ (w[i-2]  >> 10);
        w[i] = w[i-16] + s0 + w[i-7] + s1;
    }
    uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
    uint32_t e = ctx->state[4], f = ctx->state[5], g = ctx->state[6], h = ctx->state[7];
    for (int i = 0; i < 64; i++) {
        uint32_t S1   = rotr32(e, 6) ^ rotr32(e, 11) ^ rotr32(e, 25);
        uint32_t ch   = (e & f) ^ (~e & g);
        uint32_t temp1 = h + S1 + ch + SHA256_K[i] + w[i];
        uint32_t S0   = rotr32(a, 2) ^ rotr32(a, 13) ^ rotr32(a, 22);
        uint32_t maj  = (a & b) ^ (a & c) ^ (b & c);
        uint32_t temp2 = S0 + maj;
        h = g; g = f; f = e; e = d + temp1;
        d = c; c = b; b = a; a = temp1 + temp2;
    }
    ctx->state[0] += a; ctx->state[1] += b; ctx->state[2] += c; ctx->state[3] += d;
    ctx->state[4] += e; ctx->state[5] += f; ctx->state[6] += g; ctx->state[7] += h;
}

static void sha256_update(SHA256Ctx *ctx, const uint8_t *data, size_t len) {
    ctx->count += (uint64_t)len * 8;
    while (len > 0) {
        uint32_t space = 64 - ctx->buf_len;
        uint32_t take  = (uint32_t)(len < space ? len : space);
        memcpy(ctx->buf + ctx->buf_len, data, take);
        ctx->buf_len += take;
        data += take;
        len  -= take;
        if (ctx->buf_len == 64) {
            sha256_transform(ctx, ctx->buf);
            ctx->buf_len = 0;
        }
    }
}

static void sha256_final(SHA256Ctx *ctx, uint8_t digest[32]) {
    // Append padding bit.
    ctx->buf[ctx->buf_len++] = 0x80;
    if (ctx->buf_len > 56) {
        memset(ctx->buf + ctx->buf_len, 0, 64 - ctx->buf_len);
        sha256_transform(ctx, ctx->buf);
        ctx->buf_len = 0;
    }
    memset(ctx->buf + ctx->buf_len, 0, 56 - ctx->buf_len);
    // Append bit-length (big-endian).
    for (int i = 0; i < 8; i++) {
        ctx->buf[56 + i] = (uint8_t)(ctx->count >> (56 - 8*i));
    }
    sha256_transform(ctx, ctx->buf);
    for (int i = 0; i < 8; i++) {
        digest[i*4+0] = (uint8_t)(ctx->state[i] >> 24);
        digest[i*4+1] = (uint8_t)(ctx->state[i] >> 16);
        digest[i*4+2] = (uint8_t)(ctx->state[i] >>  8);
        digest[i*4+3] = (uint8_t)(ctx->state[i]);
    }
}

// ─── AES-256 key schedule + block encrypt ────────────────────────────────────

static const uint8_t SBOX[256] = {
    0x63,0x7c,0x77,0x7b,0xf2,0x6b,0x6f,0xc5,0x30,0x01,0x67,0x2b,0xfe,0xd7,0xab,0x76,
    0xca,0x82,0xc9,0x7d,0xfa,0x59,0x47,0xf0,0xad,0xd4,0xa2,0xaf,0x9c,0xa4,0x72,0xc0,
    0xb7,0xfd,0x93,0x26,0x36,0x3f,0xf7,0xcc,0x34,0xa5,0xe5,0xf1,0x71,0xd8,0x31,0x15,
    0x04,0xc7,0x23,0xc3,0x18,0x96,0x05,0x9a,0x07,0x12,0x80,0xe2,0xeb,0x27,0xb2,0x75,
    0x09,0x83,0x2c,0x1a,0x1b,0x6e,0x5a,0xa0,0x52,0x3b,0xd6,0xb3,0x29,0xe3,0x2f,0x84,
    0x53,0xd1,0x00,0xed,0x20,0xfc,0xb1,0x5b,0x6a,0xcb,0xbe,0x39,0x4a,0x4c,0x58,0xcf,
    0xd0,0xef,0xaa,0xfb,0x43,0x4d,0x33,0x85,0x45,0xf9,0x02,0x7f,0x50,0x3c,0x9f,0xa8,
    0x51,0xa3,0x40,0x8f,0x92,0x9d,0x38,0xf5,0xbc,0xb6,0xda,0x21,0x10,0xff,0xf3,0xd2,
    0xcd,0x0c,0x13,0xec,0x5f,0x97,0x44,0x17,0xc4,0xa7,0x7e,0x3d,0x64,0x5d,0x19,0x73,
    0x60,0x81,0x4f,0xdc,0x22,0x2a,0x90,0x88,0x46,0xee,0xb8,0x14,0xde,0x5e,0x0b,0xdb,
    0xe0,0x32,0x3a,0x0a,0x49,0x06,0x24,0x5c,0xc2,0xd3,0xac,0x62,0x91,0x95,0xe4,0x79,
    0xe7,0xc8,0x37,0x6d,0x8d,0xd5,0x4e,0xa9,0x6c,0x56,0xf4,0xea,0x65,0x7a,0xae,0x08,
    0xba,0x78,0x25,0x2e,0x1c,0xa6,0xb4,0xc6,0xe8,0xdd,0x74,0x1f,0x4b,0xbd,0x8b,0x8a,
    0x70,0x3e,0xb5,0x66,0x48,0x03,0xf6,0x0e,0x61,0x35,0x57,0xb9,0x86,0xc1,0x1d,0x9e,
    0xe1,0xf8,0x98,0x11,0x69,0xd9,0x8e,0x94,0x9b,0x1e,0x87,0xe9,0xce,0x55,0x28,0xdf,
    0x8c,0xa1,0x89,0x0d,0xbf,0xe6,0x42,0x68,0x41,0x99,0x2d,0x0f,0xb0,0x54,0xbb,0x16,
};

static const uint8_t INV_SBOX[256] = {
    0x52,0x09,0x6a,0xd5,0x30,0x36,0xa5,0x38,0xbf,0x40,0xa3,0x9e,0x81,0xf3,0xd7,0xfb,
    0x7c,0xe3,0x39,0x82,0x9b,0x2f,0xff,0x87,0x34,0x8e,0x43,0x44,0xc4,0xde,0xe9,0xcb,
    0x54,0x7b,0x94,0x32,0xa6,0xc2,0x23,0x3d,0xee,0x4c,0x95,0x0b,0x42,0xfa,0xc3,0x4e,
    0x08,0x2e,0xa1,0x66,0x28,0xd9,0x24,0xb2,0x76,0x5b,0xa2,0x49,0x6d,0x8b,0xd1,0x25,
    0x72,0xf8,0xf6,0x64,0x86,0x68,0x98,0x16,0xd4,0xa4,0x5c,0xcc,0x5d,0x65,0xb6,0x92,
    0x6c,0x70,0x48,0x50,0xfd,0xed,0xb9,0xda,0x5e,0x15,0x46,0x57,0xa7,0x8d,0x9d,0x84,
    0x90,0xd8,0xab,0x00,0x8c,0xbc,0xd3,0x0a,0xf7,0xe4,0x58,0x05,0xb8,0xb3,0x45,0x06,
    0xd0,0x2c,0x1e,0x8f,0xca,0x3f,0x0f,0x02,0xc1,0xaf,0xbd,0x03,0x01,0x13,0x8a,0x6b,
    0x3a,0x91,0x11,0x41,0x4f,0x67,0xdc,0xea,0x97,0xf2,0xcf,0xce,0xf0,0xb4,0xe6,0x73,
    0x96,0xac,0x74,0x22,0xe7,0xad,0x35,0x85,0xe2,0xf9,0x37,0xe8,0x1c,0x75,0xdf,0x6e,
    0x47,0xf1,0x1a,0x71,0x1d,0x29,0xc5,0x89,0x6f,0xb7,0x62,0x0e,0xaa,0x18,0xbe,0x1b,
    0xfc,0x56,0x3e,0x4b,0xc6,0xd2,0x79,0x20,0x9a,0xdb,0xc0,0xfe,0x78,0xcd,0x5a,0xf4,
    0x1f,0xdd,0xa8,0x33,0x88,0x07,0xc7,0x31,0xb1,0x12,0x10,0x59,0x27,0x80,0xec,0x5f,
    0x60,0x51,0x7f,0xa9,0x19,0xb5,0x4a,0x0d,0x2d,0xe5,0x7a,0x9f,0x93,0xc9,0x9c,0xef,
    0xa0,0xe0,0x3b,0x4d,0xae,0x2a,0xf5,0xb0,0xc8,0xeb,0xbb,0x3c,0x83,0x53,0x99,0x61,
    0x17,0x2b,0x04,0x7e,0xba,0x77,0xd6,0x26,0xe1,0x69,0x14,0x63,0x55,0x21,0x0c,0x7d,
};

static inline uint8_t xtime(uint8_t x) {
    return (uint8_t)((x << 1) ^ ((x >> 7) * 0x1b));
}
static inline uint8_t gmul(uint8_t a, uint8_t b) {
    uint8_t p = 0;
    for (int i = 0; i < 8; i++) {
        if (b & 1) p ^= a;
        bool hb = (a & 0x80) != 0;
        a <<= 1;
        if (hb) a ^= 0x1b;
        b >>= 1;
    }
    return p;
}

struct AES256Ctx {
    uint8_t rk[240]; // 15 round keys × 16 bytes
};

static void aes256_key_expand(AES256Ctx *ctx, const uint8_t key[32]) {
    static const uint8_t rcon[11] = {
        0x00,0x01,0x02,0x04,0x08,0x10,0x20,0x40,0x80,0x1b,0x36
    };
    memcpy(ctx->rk, key, 32);
    for (int i = 8; i < 60; i++) {
        uint8_t tmp[4];
        memcpy(tmp, ctx->rk + (i-1)*4, 4);
        if (i % 8 == 0) {
            uint8_t t = tmp[0];
            tmp[0] = SBOX[tmp[1]] ^ rcon[i/8];
            tmp[1] = SBOX[tmp[2]];
            tmp[2] = SBOX[tmp[3]];
            tmp[3] = SBOX[t];
        } else if (i % 8 == 4) {
            for (int j = 0; j < 4; j++) tmp[j] = SBOX[tmp[j]];
        }
        for (int j = 0; j < 4; j++) {
            ctx->rk[i*4+j] = ctx->rk[(i-8)*4+j] ^ tmp[j];
        }
    }
}

static void aes256_encrypt_block(const AES256Ctx *ctx, const uint8_t in[16], uint8_t out[16]) {
    uint8_t s[16];
    for (int i = 0; i < 16; i++) s[i] = in[i] ^ ctx->rk[i];

    for (int round = 1; round < 14; round++) {
        // SubBytes + ShiftRows + MixColumns
        uint8_t t[16];
        t[ 0] = SBOX[s[ 0]]; t[ 1] = SBOX[s[ 5]]; t[ 2] = SBOX[s[10]]; t[ 3] = SBOX[s[15]];
        t[ 4] = SBOX[s[ 4]]; t[ 5] = SBOX[s[ 9]]; t[ 6] = SBOX[s[14]]; t[ 7] = SBOX[s[ 3]];
        t[ 8] = SBOX[s[ 8]]; t[ 9] = SBOX[s[13]]; t[10] = SBOX[s[ 2]]; t[11] = SBOX[s[ 7]];
        t[12] = SBOX[s[12]]; t[13] = SBOX[s[ 1]]; t[14] = SBOX[s[ 6]]; t[15] = SBOX[s[11]];
        for (int c = 0; c < 4; c++) {
            uint8_t *col = &t[c*4];
            uint8_t a0 = col[0], a1 = col[1], a2 = col[2], a3 = col[3];
            col[0] = gmul(a0,2)^gmul(a1,3)^a2^a3;
            col[1] = a0^gmul(a1,2)^gmul(a2,3)^a3;
            col[2] = a0^a1^gmul(a2,2)^gmul(a3,3);
            col[3] = gmul(a0,3)^a1^a2^gmul(a3,2);
        }
        const uint8_t *rk = ctx->rk + round*16;
        for (int i = 0; i < 16; i++) s[i] = t[i] ^ rk[i];
    }
    // Final round (no MixColumns)
    uint8_t t[16];
    t[ 0] = SBOX[s[ 0]]; t[ 1] = SBOX[s[ 5]]; t[ 2] = SBOX[s[10]]; t[ 3] = SBOX[s[15]];
    t[ 4] = SBOX[s[ 4]]; t[ 5] = SBOX[s[ 9]]; t[ 6] = SBOX[s[14]]; t[ 7] = SBOX[s[ 3]];
    t[ 8] = SBOX[s[ 8]]; t[ 9] = SBOX[s[13]]; t[10] = SBOX[s[ 2]]; t[11] = SBOX[s[ 7]];
    t[12] = SBOX[s[12]]; t[13] = SBOX[s[ 1]]; t[14] = SBOX[s[ 6]]; t[15] = SBOX[s[11]];
    const uint8_t *rk = ctx->rk + 14*16;
    for (int i = 0; i < 16; i++) out[i] = t[i] ^ rk[i];
}

// ─── GHASH for GCM ───────────────────────────────────────────────────────────

static void ghash_mul(uint8_t Z[16], const uint8_t X[16], const uint8_t H[16]) {
    uint8_t V[16];
    memcpy(V, H, 16);
    uint8_t res[16] = {};
    for (int i = 0; i < 16; i++) {
        for (int bit = 7; bit >= 0; bit--) {
            if ((X[i] >> bit) & 1) {
                for (int j = 0; j < 16; j++) res[j] ^= V[j];
            }
            bool lsb = (V[15] & 1) != 0;
            uint8_t carry = 0;
            for (int j = 0; j < 16; j++) {
                uint8_t next = V[j] >> 1;
                next |= carry;
                carry = (V[j] & 1) ? 0x80 : 0;
                V[j] = next;
            }
            if (lsb) V[0] ^= 0xe1;
        }
    }
    memcpy(Z, res, 16);
}

static void ghash(const uint8_t H[16], const uint8_t *data, size_t len, uint8_t tag[16]) {
    uint8_t Y[16] = {};
    for (size_t i = 0; i < len; i += 16) {
        uint8_t block[16] = {};
        size_t take = (len - i) < 16 ? (len - i) : 16;
        memcpy(block, data + i, take);
        for (int j = 0; j < 16; j++) Y[j] ^= block[j];
        ghash_mul(Y, Y, H);
    }
    memcpy(tag, Y, 16);
}

// Increment the AES-CTR counter block (big-endian, last 4 bytes).
static void ctr_inc(uint8_t ctr[16]) {
    for (int i = 15; i >= 12; i--) {
        if (++ctr[i]) break;
    }
}

} // anonymous namespace

// ─── Public C interface ───────────────────────────────────────────────────────

#ifdef __cplusplus
extern "C" {
#endif

// SHA-256 digest of arbitrary data.
void ntl_sha256(const uint8_t *data, size_t len, uint8_t digest[32]) {
    SHA256Ctx ctx;
    sha256_init(&ctx);
    sha256_update(&ctx, data, len);
    sha256_final(&ctx, digest);
}

// HMAC-SHA-256 (RFC 2104).
void ntl_hmac_sha256(const uint8_t *key, size_t klen,
                     const uint8_t *data, size_t dlen,
                     uint8_t digest[32]) {
    uint8_t k[64] = {};
    if (klen > 64) {
        // Hash long keys.
        ntl_sha256(key, klen, k);
    } else {
        memcpy(k, key, klen);
    }
    uint8_t ipad[64], opad[64];
    for (int i = 0; i < 64; i++) {
        ipad[i] = k[i] ^ 0x36;
        opad[i] = k[i] ^ 0x5c;
    }
    SHA256Ctx ctx;
    sha256_init(&ctx);
    sha256_update(&ctx, ipad, 64);
    sha256_update(&ctx, data, dlen);
    uint8_t inner[32];
    sha256_final(&ctx, inner);

    sha256_init(&ctx);
    sha256_update(&ctx, opad, 64);
    sha256_update(&ctx, inner, 32);
    sha256_final(&ctx, digest);
}

// AES-256-GCM encrypt. Returns 0 on success, -1 on error.
// key: 32 bytes, nonce: 12 bytes, ct and tag are written by the function.
// ct must be at least ptlen bytes.
int ntl_aes256gcm_encrypt(const uint8_t key[32], const uint8_t nonce[12],
                           const uint8_t *pt, size_t ptlen,
                           uint8_t *ct, uint8_t tag[16]) {
    AES256Ctx aes;
    aes256_key_expand(&aes, key);

    // Derive H = AES(0^128).
    uint8_t H[16] = {};
    aes256_encrypt_block(&aes, H, H);

    // Build J0 (counter block).
    uint8_t J0[16] = {};
    memcpy(J0, nonce, 12);
    J0[15] = 1;

    // CTR encrypt.
    uint8_t ctr[16];
    memcpy(ctr, J0, 16);
    ctr_inc(ctr);

    for (size_t i = 0; i < ptlen; i += 16) {
        uint8_t ks[16];
        aes256_encrypt_block(&aes, ctr, ks);
        ctr_inc(ctr);
        size_t take = (ptlen - i) < 16 ? (ptlen - i) : 16;
        for (size_t j = 0; j < take; j++) ct[i+j] = pt[i+j] ^ ks[j];
    }

    // Compute auth tag: GHASH(H, ct) XOR E(J0).
    uint8_t ghash_buf[16] = {};
    // Pack lengths (A=0 bits, C=ptlen*8 bits) as 16-byte big-endian.
    uint8_t lenblock[16] = {};
    uint64_t clen = (uint64_t)ptlen * 8;
    for (int i = 0; i < 8; i++) lenblock[8+i] = (uint8_t)(clen >> (56-8*i));

    // GHASH over (ct ‖ len_block).
    uint8_t S[16] = {};
    for (size_t i = 0; i < ptlen; i += 16) {
        uint8_t block[16] = {};
        size_t take = (ptlen - i) < 16 ? (ptlen - i) : 16;
        memcpy(block, ct + i, take);
        for (int j = 0; j < 16; j++) S[j] ^= block[j];
        ghash_mul(S, S, H);
    }
    for (int j = 0; j < 16; j++) S[j] ^= lenblock[j];
    ghash_mul(S, S, H);

    uint8_t EJ0[16];
    aes256_encrypt_block(&aes, J0, EJ0);
    for (int i = 0; i < 16; i++) tag[i] = S[i] ^ EJ0[i];
    return 0;
}

// AES-256-GCM decrypt. Returns 0 on success (tag match), -1 on tag mismatch.
// pt must be at least ctlen bytes.
int ntl_aes256gcm_decrypt(const uint8_t key[32], const uint8_t nonce[12],
                           const uint8_t *ct, size_t ctlen,
                           const uint8_t expected_tag[16], uint8_t *pt) {
    uint8_t computed_tag[16];
    // Encrypt produces same keystream; decrypt = encrypt for CTR mode.
    // But first verify the tag.
    AES256Ctx aes;
    aes256_key_expand(&aes, key);
    uint8_t H[16] = {};
    aes256_encrypt_block(&aes, H, H);
    uint8_t J0[16] = {};
    memcpy(J0, nonce, 12);
    J0[15] = 1;

    // Compute expected tag from ciphertext.
    uint8_t S[16] = {};
    uint64_t clen = (uint64_t)ctlen * 8;
    uint8_t lenblock[16] = {};
    for (int i = 0; i < 8; i++) lenblock[8+i] = (uint8_t)(clen >> (56-8*i));
    for (size_t i = 0; i < ctlen; i += 16) {
        uint8_t block[16] = {};
        size_t take = (ctlen - i) < 16 ? (ctlen - i) : 16;
        memcpy(block, ct + i, take);
        for (int j = 0; j < 16; j++) S[j] ^= block[j];
        ghash_mul(S, S, H);
    }
    for (int j = 0; j < 16; j++) S[j] ^= lenblock[j];
    ghash_mul(S, S, H);

    uint8_t EJ0[16];
    aes256_encrypt_block(&aes, J0, EJ0);
    uint8_t diff = 0;
    for (int i = 0; i < 16; i++) {
        computed_tag[i] = S[i] ^ EJ0[i];
        diff |= computed_tag[i] ^ expected_tag[i];
    }
    if (diff != 0) return -1; // Tag mismatch — authentication failure.

    // CTR decrypt.
    uint8_t ctr[16];
    memcpy(ctr, J0, 16);
    ctr_inc(ctr);
    for (size_t i = 0; i < ctlen; i += 16) {
        uint8_t ks[16];
        aes256_encrypt_block(&aes, ctr, ks);
        ctr_inc(ctr);
        size_t take = (ctlen - i) < 16 ? (ctlen - i) : 16;
        for (size_t j = 0; j < take; j++) pt[i+j] = ct[i+j] ^ ks[j];
    }
    return 0;
}

// Convenience: write SHA-256 as a 64-char hex string (+ NUL terminator).
void ntl_sha256_hex(const uint8_t *data, size_t len, char out[65]) {
    uint8_t digest[32];
    ntl_sha256(data, len, digest);
    static const char hex[] = "0123456789abcdef";
    for (int i = 0; i < 32; i++) {
        out[i*2+0] = hex[digest[i] >> 4];
        out[i*2+1] = hex[digest[i] & 0xf];
    }
    out[64] = '\0';
}

#ifdef __cplusplus
} // extern "C"
#endif
