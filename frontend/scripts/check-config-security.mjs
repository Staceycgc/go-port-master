import { sanitizeRemoteHosts, containsCredentialFields } from '../src/utils/remoteHostsSanitize.js'

function assert(condition, message) {
  if (!condition) {
    console.error('FAIL:', message)
    process.exit(1)
  }
}

const sanitized = sanitizeRemoteHosts([
  {
    id: 'h1',
    name: 'prod',
    host: '10.0.0.1',
    port: 22,
    username: 'root',
    authType: 'password',
    credential: 'secret',
    password: 'secret',
    privateKey: 'pem'
  },
  { host: '', username: 'x' },
  null
])

assert(sanitized.length === 1, 'expected one sanitized host')
assert(sanitized[0].host === '10.0.0.1', 'host preserved')
assert(!('credential' in sanitized[0]), 'credential stripped')
assert(!('password' in sanitized[0]), 'password stripped')
assert(!containsCredentialFields(sanitized), 'sanitized output has no credentials')

const exported = sanitizeRemoteHosts([
  { id: 'x', name: 'x', host: '1.2.3.4', port: 22, username: 'u', authType: 'key', privateKey: 'x' }
])
assert(!containsCredentialFields(exported), 'export sanitize removes privateKey')

console.log('Config security checks passed')
