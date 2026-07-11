import * as XLSX from 'xlsx'
import { saveAs } from 'file-saver'

export function buildExportColumns(t) {
  return {
    protocol: t('table.protocol'),
    port: t('table.port'),
    localAddress: t('table.localAddress'),
    foreignAddress: t('table.foreignAddress'),
    pid: t('table.pid'),
    processName: t('table.processName'),
    programPath: t('export.programPath'),
    state: t('table.state'),
    sheetName: t('export.sheetName')
  }
}

export function exportToExcel(data, filename = 'ports', columns) {
  const rows = data.map(item => ({
    [columns.protocol]: item.protocol,
    [columns.port]: item.port,
    [columns.localAddress]: item.localAddress,
    [columns.foreignAddress]: item.foreignAddress,
    [columns.pid]: item.pid || '',
    [columns.processName]: item.processName,
    [columns.programPath]: item.programPath,
    [columns.state]: item.state
  }))
  const ws = XLSX.utils.json_to_sheet(rows)
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, columns.sheetName)
  const buf = XLSX.write(wb, { bookType: 'xlsx', type: 'array' })
  saveAs(new Blob([buf], { type: 'application/octet-stream' }), `${filename}.xlsx`)
}

export function exportToMarkdown(data, filename = 'ports', columns) {
  const header = `| ${columns.protocol} | ${columns.port} | ${columns.localAddress} | ${columns.foreignAddress} | ${columns.pid} | ${columns.processName} | ${columns.programPath} | ${columns.state} |\n`
  const sep = '| --- | --- | --- | --- | --- | --- | --- | --- |\n'
  const rows = data.map(item =>
    `| ${item.protocol} | ${item.port} | ${item.localAddress} | ${item.foreignAddress} | ${item.pid || ''} | ${item.processName} | ${item.programPath} | ${item.state} |`
  ).join('\n')
  const content = header + sep + rows
  saveAs(new Blob([content], { type: 'text/markdown;charset=utf-8' }), `${filename}.md`)
}

export function exportToTxt(data, filename = 'ports', columns) {
  const header = [columns.protocol, columns.port, columns.localAddress, columns.foreignAddress,
    columns.pid, columns.processName, columns.programPath, columns.state].join('\t') + '\n'
  const rows = data.map(item =>
    [item.protocol, item.port, item.localAddress, item.foreignAddress,
     item.pid || '', item.processName, item.programPath, item.state].join('\t')
  ).join('\n')
  saveAs(new Blob([header + rows], { type: 'text/plain;charset=utf-8' }), `${filename}.txt`)
}
