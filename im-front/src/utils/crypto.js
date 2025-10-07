// 加密相关工具函数

/**
 * 生成DID地址
 * @param {string} publicKey - 公钥
 * @returns {string} DID地址
 */
export function generateDID(publicKey) {
  // 基于公钥内容生成稳定但不固定的标识符（避免对base64字符串再次编码导致常量前缀）
  // 直接使用公钥base64的末尾部分作为标识符来源，保证不同公钥得到不同DID
  const normalized = (publicKey || '').replace(/[^A-Za-z0-9]/g, '')
  const identifier = normalized.length >= 32
    ? normalized.slice(-32)
    : (normalized + '0'.repeat(32 - normalized.length))
  return `did:qlink:${identifier}`
}

/**
 * 验证DID格式
 * @param {string} did - DID地址
 * @returns {boolean} 是否有效
 */
export function validateDID(did) {
  const didPattern = /^did:qlink:[a-zA-Z0-9]{32}$/
  return didPattern.test(did)
}

/**
 * 生成密钥对（模拟）
 * @returns {Promise<{publicKey: string, privateKey: string}>}
 */
export async function generateKeyPair() {
  // 在实际应用中，这里应该使用真正的密码学库
  // 这里只是模拟生成
  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256"
    },
    true,
    ["sign", "verify"]
  )
  
  const publicKey = await window.crypto.subtle.exportKey("spki", keyPair.publicKey)
  const privateKey = await window.crypto.subtle.exportKey("pkcs8", keyPair.privateKey)
  // 额外导出JWK，便于构建符合后端要求的JsonWebKey2020验证方法
  const publicKeyJwk = await window.crypto.subtle.exportKey("jwk", keyPair.publicKey)
  
  return {
    publicKey: btoa(String.fromCharCode(...new Uint8Array(publicKey))),
    privateKey: btoa(String.fromCharCode(...new Uint8Array(privateKey))),
    jwk: publicKeyJwk
  }
}

/**
 * 将IEEE P1363格式的签名转换为ASN.1 DER格式
 * @param {ArrayBuffer} p1363Signature - IEEE P1363格式的签名 (r||s)
 * @returns {ArrayBuffer} ASN.1 DER格式的签名
 */
function convertP1363ToASN1(p1363Signature) {
  const signature = new Uint8Array(p1363Signature)
  const r = signature.slice(0, 32) // P-256的r和s各32字节
  const s = signature.slice(32, 64)
  
  // 构建ASN.1 DER格式
  // SEQUENCE { INTEGER r, INTEGER s }
  
  function encodeInteger(bytes) {
    // 如果最高位是1，需要在前面添加0x00以表示正数
    const needsPadding = bytes[0] >= 0x80
    const length = bytes.length + (needsPadding ? 1 : 0)
    const result = new Uint8Array(2 + length)
    
    result[0] = 0x02 // INTEGER tag
    result[1] = length
    
    if (needsPadding) {
      result[2] = 0x00
      result.set(bytes, 3)
    } else {
      result.set(bytes, 2)
    }
    
    return result
  }
  
  const rEncoded = encodeInteger(r)
  const sEncoded = encodeInteger(s)
  
  // 构建SEQUENCE
  const totalLength = rEncoded.length + sEncoded.length
  const result = new Uint8Array(2 + totalLength)
  
  result[0] = 0x30 // SEQUENCE tag
  result[1] = totalLength
  result.set(rEncoded, 2)
  result.set(sEncoded, 2 + rEncoded.length)
  
  return result.buffer
}

/**
 * 使用ECDSA签名数据
 * @param {string} data - 要签名的数据
 * @param {string} privateKeyBase64 - Base64编码的私钥
 * @returns {Promise<string>} Base64编码的ASN.1格式ECDSA签名
 */
export async function signData(data, privateKeyBase64) {
  try {
    // 将Base64私钥转换为ArrayBuffer
    const privateKeyBuffer = base64ToArrayBuffer(privateKeyBase64)
    
    // 导入私钥
    const privateKey = await window.crypto.subtle.importKey(
      'pkcs8',
      privateKeyBuffer,
      {
        name: 'ECDSA',
        namedCurve: 'P-256'
      },
      false,
      ['sign']
    )
    
    // 编码数据
    const encoder = new TextEncoder()
    const dataBuffer = encoder.encode(data)
    
    // 使用ECDSA P-256 + SHA-256进行签名
    const signatureBuffer = await window.crypto.subtle.sign(
      {
        name: 'ECDSA',
        hash: { name: 'SHA-256' }
      },
      privateKey,
      dataBuffer
    )
    
    // 将IEEE P1363格式转换为ASN.1 DER格式
    const asn1Signature = convertP1363ToASN1(signatureBuffer)
    
    // 将签名转换为Base64
    return arrayBufferToBase64(asn1Signature)
  } catch (error) {
    console.error('ECDSA签名失败:', error)
    throw new Error('ECDSA签名失败: ' + error.message)
  }
}

/**
 * 验证ECDSA签名
 * @param {string} data - 原始数据
 * @param {string} signatureBase64 - Base64编码的签名
 * @param {string} publicKeyBase64 - Base64编码的公钥
 * @returns {Promise<boolean>} 验证结果
 */
export async function verifySignature(data, signatureBase64, publicKeyBase64) {
  try {
    // 将Base64公钥转换为ArrayBuffer
    const publicKeyBuffer = base64ToArrayBuffer(publicKeyBase64)
    
    // 导入公钥
    const publicKey = await window.crypto.subtle.importKey(
      'spki',
      publicKeyBuffer,
      {
        name: 'ECDSA',
        namedCurve: 'P-256'
      },
      false,
      ['verify']
    )
    
    // 编码数据
    const encoder = new TextEncoder()
    const dataBuffer = encoder.encode(data)
    
    // 将Base64签名转换为ArrayBuffer
    const signatureBuffer = base64ToArrayBuffer(signatureBase64)
    
    // 验证ECDSA签名
    const isValid = await window.crypto.subtle.verify(
      {
        name: 'ECDSA',
        hash: { name: 'SHA-256' }
      },
      publicKey,
      signatureBuffer,
      dataBuffer
    )
    
    return isValid
  } catch (error) {
    console.error('验证ECDSA签名失败:', error)
    return false
  }
}

/**
 * 生成双密钥对（ECDSA + 格加密）
 * @returns {Promise<{ecdsaKeyPair: {publicKey: string, privateKey: string}, latticeKeyPair: {publicKey: string, privateKey: string}}>}
 */
export async function generateDualKeyPair() {
  try {
    // 生成ECDSA密钥对（用于身份验证）
    const ecdsaKeyPair = await generateKeyPair()
    
    // 生成格加密密钥对（用于通信加密）
    const latticeKeyPair = await generateKyberKeyPair()
    
    return {
      ecdsaKeyPair,
      latticeKeyPair
    }
  } catch (error) {
    console.error('生成双密钥对失败:', error)
    throw new Error('生成双密钥对失败: ' + error.message)
  }
}

/**
 * 生成真实的Kyber768密钥对（格加密）
 * @returns {Promise<{publicKey: string, privateKey: string}>}
 */
export async function generateKyberKeyPair() {
  try {
    // 注意：这里应该使用真正的Kyber768实现
    // 目前使用模拟实现，实际部署时需要集成真正的Kyber768库
    
    // 生成32字节的随机种子
    const seed = new Uint8Array(32)
    window.crypto.getRandomValues(seed)
    
    // 模拟Kyber768密钥生成
    // 实际应该调用后端API或使用WebAssembly版本的Kyber768
    const publicKeyBytes = new Uint8Array(1184) // Kyber768公钥长度
    const privateKeyBytes = new Uint8Array(2400) // Kyber768私钥长度
    
    window.crypto.getRandomValues(publicKeyBytes)
    window.crypto.getRandomValues(privateKeyBytes)
    
    const publicKey = arrayBufferToBase64(publicKeyBytes.buffer)
    const privateKey = arrayBufferToBase64(privateKeyBytes.buffer)
    
    return { publicKey, privateKey }
  } catch (error) {
    console.error('生成Kyber768密钥对失败:', error)
    throw new Error('生成Kyber768密钥对失败: ' + error.message)
  }
}

/**
 * 模拟Kyber768封装：根据对方公钥生成密文与共享密钥
 * @param {string} peerPublicKey - 对方的Kyber768公钥
 * @returns {{ciphertext: string, sharedSecret: string}}
 */
export function generateKyberEncapsulate(peerPublicKey) {
  // 真实实现应使用Kyber768封装返回 { ct, ss }
  const salt = generateChallenge()
  const base = `kyber768|encap|${peerPublicKey}|${salt}`
  const ciphertext = btoa(base).replace(/[^A-Za-z0-9]/g, '')
  // 共享密钥可由密文派生，便于Alice解封装得到一致的值
  const sharedSecret = btoa(ciphertext).substring(0, 32)
  return { ciphertext, sharedSecret }
}

/**
 * 模拟Kyber768解封装：Alice使用私钥与密文派生相同的共享密钥
 * @param {string} ciphertext - 封装密文
 * @param {string} privateKey - Alice的Kyber768私钥
 * @returns {string} sharedSecret
 */
export function generateSharedSecretFromCiphertext(ciphertext, privateKey) {
  // 真实实现应使用Kyber768解封装算法 decap(ct, sk)
  const combined = `kyber768|decap|${ciphertext}|${(privateKey||'').slice(0,16)}`
  return btoa(combined).substring(0, 32)
}

/**
 * 生成共享密钥（使用格加密Kyber768）
 * @param {string} privateKey - Kyber768私钥
 * @param {string} peerPublicKey - 对方的Kyber768公钥
 * @returns {Promise<string>} 共享密钥
 */
export async function generateSharedSecret(privateKey, peerPublicKey) {
  // TODO: 集成真正的Kyber768密钥协商算法
  // 当前为模拟实现，实际应使用Kyber768解封装算法
  // 确保与后端的格加密实现保持一致
  const combined = 'kyber768:' + privateKey + ':' + peerPublicKey
  const hash = btoa(combined).substring(0, 32)
  return hash
}

/**
 * AES加密（使用格加密派生的密钥）
 * @param {string} plaintext - 明文
 * @param {string} key - 密钥（来自Kyber768密钥交换）
 * @returns {Promise<string>} 密文
 */
export async function encryptMessage(plaintext, key) {
  try {
    // 使用SHA-256对共享密钥进行扩展，得到32字节AES-256密钥
    const enc = new TextEncoder()
    const keyMaterial = await window.crypto.subtle.digest('SHA-256', enc.encode(key))
    const aesKey = await window.crypto.subtle.importKey(
      'raw',
      keyMaterial,
      { name: 'AES-GCM' },
      false,
      ['encrypt']
    )

    // 生成12字节的随机nonce（GCM要求）
    const nonceBytes = new Uint8Array(12)
    window.crypto.getRandomValues(nonceBytes)

    // 加密明文
    const ciphertextBuf = await window.crypto.subtle.encrypt(
      { name: 'AES-GCM', iv: nonceBytes },
      aesKey,
      enc.encode(plaintext)
    )

    return {
      ciphertext: arrayBufferToBase64(ciphertextBuf),
      nonce: arrayBufferToBase64(nonceBytes.buffer)
    }
  } catch (error) {
    console.error('格加密消息加密失败:', error)
    throw new Error('格加密消息加密失败')
  }
}

/**
 * AES解密（使用格加密派生的密钥）
 * @param {string} ciphertext - 密文
 * @param {string} key - 密钥（来自Kyber768密钥交换）
 * @returns {Promise<string>} 明文
 */
export async function decryptMessage(ciphertext, nonce, key) {
  try {
    const enc = new TextEncoder()
    const keyMaterial = await window.crypto.subtle.digest('SHA-256', enc.encode(key))
    const aesKey = await window.crypto.subtle.importKey(
      'raw',
      keyMaterial,
      { name: 'AES-GCM' },
      false,
      ['decrypt']
    )

    const ctBuf = base64ToArrayBuffer(ciphertext)
    const nonceBuf = base64ToArrayBuffer(nonce)

    const plainBuf = await window.crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: new Uint8Array(nonceBuf) },
      aesKey,
      ctBuf
    )
    const decoder = new TextDecoder()
    return decoder.decode(plainBuf)
  } catch (error) {
    console.error('格加密消息解密失败:', error)
    throw new Error('格加密消息解密失败')
  }
}

/**
 * 生成随机挑战
 * @returns {string} 随机挑战字符串
 */
export function generateChallenge() {
  const randomBytes = new Uint8Array(16)
  window.crypto.getRandomValues(randomBytes)
  return btoa(String.fromCharCode(...randomBytes)).replace(/[+/=]/g, '')
}

/**
 * 存储密钥到本地存储
 * @param {string} key - 密钥名称
 * @param {string} value - 密钥值
 */
export function storeKey(key, value) {
  try {
    localStorage.setItem(`qlink_key_${key}`, value)
  } catch (error) {
    console.error('存储密钥失败:', error)
  }
}

/**
 * 从本地存储获取密钥
 * @param {string} key - 密钥名称
 * @returns {string|null} 密钥值
 */
export function getStoredKey(key) {
  try {
    return localStorage.getItem(`qlink_key_${key}`)
  } catch (error) {
    console.error('获取密钥失败:', error)
    return null
  }
}

/**
 * 生成ECDSA签名（用于质询响应）
 * @param {string} challenge - 质询字符串
 * @param {string} privateKeyBase64 - Base64编码的ECDSA私钥
 * @returns {Promise<string>} Base64编码的混合签名JSON
 */
export async function generateECDSASignature(challenge, privateKeyBase64) {
  try {
    // 使用ECDSA签名质询
    const ecdsaSignature = await signData(challenge, privateKeyBase64)
    
    // 创建HybridSignature格式的对象
    const hybridSignature = {
      ecdsa_signature: ecdsaSignature,
      kyber_proof: null // 质询验证只使用ECDSA，不使用Kyber
    }
    
    // 将签名对象序列化为JSON并进行base64编码
    const signatureJSON = JSON.stringify(hybridSignature)
    return btoa(signatureJSON)
    
  } catch (error) {
    console.error('生成ECDSA质询签名失败:', error)
    throw new Error('生成ECDSA质询签名失败: ' + error.message)
  }
}

/**
 * Base64转ArrayBuffer辅助函数
 * @param {string} base64 - Base64字符串
 * @returns {ArrayBuffer} ArrayBuffer
 */
function base64ToArrayBuffer(base64) {
  const binaryString = atob(base64)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes.buffer
}

/**
  * ArrayBuffer转Base64辅助函数
  * @param {ArrayBuffer} buffer - ArrayBuffer
  * @returns {string} Base64字符串
  */
 function arrayBufferToBase64(buffer) {
   const bytes = new Uint8Array(buffer)
   let binary = ''
   for (let i = 0; i < bytes.byteLength; i++) {
     binary += String.fromCharCode(bytes[i])
   }
   return btoa(binary)
 }

/**
 * 删除存储的密钥
 * @param {string} key - 密钥名称
 */
export function removeStoredKey(key) {
  try {
    localStorage.removeItem(`qlink_key_${key}`)
  } catch (error) {
    console.error('删除密钥失败:', error)
  }
}