type Level = 'debug' | 'info' | 'warn' | 'error'

const prod = import.meta.env.PROD

function emit(level: Level, msg: string, extra?: unknown) {
  if (prod && level === 'debug') return
  const row = { t: new Date().toISOString(), level, msg, extra }
  if (level === 'error') console.error(JSON.stringify(row))
  else if (level === 'warn') console.warn(JSON.stringify(row))
  else if (!prod) console.info(JSON.stringify(row))
}

export const log = {
  debug: (m: string, e?: unknown) => emit('debug', m, e),
  info: (m: string, e?: unknown) => emit('info', m, e),
  warn: (m: string, e?: unknown) => emit('warn', m, e),
  error: (m: string, e?: unknown) => emit('error', m, e),
}
