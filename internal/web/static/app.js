"use strict";

/* ------------------------------------------------------------------ */
/* constants                                                          */
/* ------------------------------------------------------------------ */

const NAV = [
  { id: "overview", label: "◉  控制台概览" },
  { id: "system", label: "▤  系统服务" },
  {
    id: "metadata",
    label: "⬡  元数据",
    children: [
      ["device", "设备"],
      ["service", "设备服务"],
      ["profile", "设备模型"],
      ["watcher", "自动发现"],
    ],
  },
  {
    id: "data",
    label: "▥  数据中心",
    children: [
      ["event", "事件"],
      ["reading", "读数"],
    ],
  },
  { id: "mappings", label: "⇄  映射管理" },
  { id: "settings", label: "⚙  连接设置" },
];

const META_LABEL = { device: "设备", service: "设备服务", profile: "设备模型", watcher: "自动发现" };

const META_COLUMNS = {
  device: ["name", "description", "serviceName", "profileName", "adminState", "operatingState"],
  service: ["name", "baseAddress", "adminState", "labels"],
  profile: ["name", "manufacturer", "model", "description"],
  watcher: ["name", "serviceName", "adminState"],
};

const META_FIELDS = {
  device: [
    { key: "name", label: "设备名称", type: "text" },
    { key: "description", label: "描述", type: "text" },
    { key: "serviceName", label: "设备服务名称", type: "text" },
    { key: "profileName", label: "设备模型名称", type: "text" },
    { key: "adminState", label: "管理状态", type: "select", options: ["UNLOCKED", "LOCKED"] },
    { key: "operatingState", label: "运行状态", type: "select", options: ["UP", "DOWN"] },
    { key: "labels", label: "标签（逗号分隔）", type: "csv" },
    { key: "protocols", label: "协议参数 protocols（JSON）", type: "json", fallback: {} },
    { key: "autoEvents", label: "自动采集 autoEvents（JSON 数组）", type: "json", fallback: [] },
  ],
  service: [
    { key: "name", label: "服务名称", type: "text" },
    { key: "description", label: "描述", type: "text" },
    { key: "baseAddress", label: "服务地址", type: "text" },
    { key: "adminState", label: "管理状态", type: "select", options: ["UNLOCKED", "LOCKED"] },
    { key: "labels", label: "标签（逗号分隔）", type: "csv" },
  ],
  profile: [
    { key: "name", label: "模型名称", type: "text" },
    { key: "description", label: "描述", type: "text" },
    { key: "manufacturer", label: "厂商", type: "text" },
    { key: "model", label: "型号", type: "text" },
    { key: "deviceResources", label: "设备资源 deviceResources（JSON 数组）", type: "json", fallback: [] },
    { key: "deviceCommands", label: "设备命令 deviceCommands（JSON 数组）", type: "json", fallback: [] },
  ],
  watcher: [
    { key: "name", label: "名称", type: "text" },
    { key: "serviceName", label: "设备服务", type: "text" },
    { key: "adminState", label: "管理状态", type: "select", options: ["UNLOCKED", "LOCKED"] },
    { key: "identifiers", label: "匹配标识 identifiers（JSON）", type: "json", fallback: {} },
    { key: "blockingIdentifiers", label: "排除标识 blockingIdentifiers（JSON）", type: "json", fallback: {} },
    {
      key: "discoveredDevice",
      label: "发现设备 discoveredDevice（JSON）",
      type: "json",
      fallback: { profileName: "", adminState: "UNLOCKED" },
    },
  ],
};

const MAPPING_FIELDS = [
  { key: "northDeviceName", label: "北向设备", type: "text" },
  { key: "northProfileName", label: "北向模型", type: "text" },
  { key: "northResourceName", label: "北向资源", type: "text" },
  { key: "southDeviceName", label: "南向设备", type: "text" },
  { key: "southProfileName", label: "南向模型", type: "text" },
  { key: "southResourceName", label: "南向资源", type: "text" },
  { key: "metadata", label: "元数据 metadata（JSON）", type: "json", fallback: {} },
];

/* ------------------------------------------------------------------ */
/* small helpers                                                      */
/* ------------------------------------------------------------------ */

const $ = (selector) => document.querySelector(selector);

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === null) continue;
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else if (key.startsWith("on") && typeof value === "function") {
      node.addEventListener(key.slice(2), value);
    } else if (key === "value") node.value = value;
    else node.setAttribute(key, value);
  }
  for (const child of [].concat(children)) {
    if (child === null || child === undefined || child === false) continue;
    node.append(child.nodeType ? child : document.createTextNode(String(child)));
  }
  return node;
}

function clear(node) {
  while (node.firstChild) node.removeChild(node.firstChild);
  return node;
}

function formatCell(value) {
  if (value === undefined || value === null) return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function table(columns, rows, actions) {
  const head = el("thead", {}, el("tr", {}, [
    ...columns.map((column) => el("th", { text: column })),
    ...(actions ? [el("th", { text: "操作" })] : []),
  ]));
  const body = el(
    "tbody",
    {},
    rows.map((row) =>
      el("tr", {}, [
        ...columns.map((column) => el("td", { text: formatCell(row[column]) })),
        ...(actions ? [el("td", {}, el("div", { class: "actions" }, actions(row)))] : []),
      ]),
    ),
  );
  return el("table", { class: "data" }, [head, body]);
}

function toast(message, type = "") {
  const node = el("div", { class: "toast " + type, text: message });
  $("#toasts").append(node);
  setTimeout(() => node.remove(), 5200);
}

function setTitle(title, breadcrumb) {
  $("#page-title").textContent = title;
  $("#breadcrumb").textContent = breadcrumb || "EdgeX Foundry";
}

async function api(path, options = {}) {
  const init = { method: options.method || "GET" };
  if (options.body !== undefined) {
    init.headers = { "Content-Type": "application/json" };
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch("/api" + path, init);
  const text = await response.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch (_) {
    data = text;
  }
  if (!response.ok) {
    const message = data && data.error ? data.error : "HTTP " + response.status;
    throw new Error(message);
  }
  return data;
}

/* ------------------------------------------------------------------ */
/* modal                                                              */
/* ------------------------------------------------------------------ */

function openModal(title, content) {
  $("#modal-title").textContent = title;
  clear($("#modal-content")).append(content);
  $("#modal").classList.remove("hidden");
}

function closeModal() {
  $("#modal").classList.add("hidden");
  clear($("#modal-content"));
}

$("#modal-close").addEventListener("click", closeModal);
$("#modal").addEventListener("click", (event) => {
  if (event.target.id === "modal") closeModal();
});

function buildForm(fields, record = {}, lockedKey = null) {
  const form = el("form", { class: "form" });
  const inputs = {};
  for (const field of fields) {
    const value = record[field.key];
    let input;
    if (field.type === "select") {
      input = el("select");
      for (const option of field.options) input.append(el("option", { value: option, text: option }));
      input.value = value || field.options[0];
    } else if (field.type === "json") {
      input = el("textarea", { rows: 6 });
      input.value =
        value === undefined
          ? JSON.stringify(field.fallback === undefined ? {} : field.fallback, null, 2)
          : JSON.stringify(value, null, 2);
    } else if (field.type === "csv") {
      input = el("input", { type: "text" });
      input.value = Array.isArray(value) ? value.join(", ") : value || "";
    } else {
      input = el("input", { type: "text" });
      input.value = value === undefined || value === null ? "" : String(value);
    }
    if (lockedKey && field.key === lockedKey) input.disabled = true;
    inputs[field.key] = { field, input };
    form.append(el("label", { class: "field" }, [el("span", { text: field.label }), input]));
  }
  return { form, inputs };
}

function collectForm(inputs, base = {}) {
  const record = { ...base };
  for (const { field, input } of Object.values(inputs)) {
    if (input.disabled) continue;
    if (field.type === "json") {
      const raw = input.value.trim();
      record[field.key] = raw ? JSON.parse(raw) : (field.fallback === undefined ? {} : field.fallback);
    } else if (field.type === "csv") {
      record[field.key] = input.value.split(",").map((part) => part.trim()).filter(Boolean);
    } else {
      record[field.key] = input.value;
    }
  }
  return record;
}

/* ------------------------------------------------------------------ */
/* navigation + router                                                */
/* ------------------------------------------------------------------ */

function renderNav(section, sub) {
  const nav = clear($("#nav"));
  for (const item of NAV) {
    if (!item.children) {
      nav.append(
        el("a", {
          href: "#/" + item.id,
          class: section === item.id ? "active" : "",
          text: item.label,
        }),
      );
      continue;
    }
    nav.append(el("div", { class: "nav-group-label", text: item.label.replace(/^\S+\s+/, "") }));
    for (const [id, label] of item.children) {
      nav.append(
        el("a", {
          href: "#/" + item.id + "/" + id,
          class: section === item.id && sub === id ? "active child" : "child",
          text: label,
        }),
      );
    }
  }
}

async function refreshConnection() {
  try {
    const health = await api("/health");
    const rows = health.services || [];
    const online = rows.filter((row) => row.status === "在线").length;
    const bad = rows.filter((row) => row.status !== "在线").map((row) => row.service);
    $("#conn").innerHTML = "";
    $("#conn").append(
      el("span", { class: online === rows.length ? "tag" : "tag bad", text: `在线 ${online}/${rows.length}` }),
      bad.length ? el("div", { class: "muted", text: "不可用：" + bad.join(", ") }) : el("div", { class: "muted", text: "全部服务正常" }),
    );
  } catch (error) {
    $("#conn").innerHTML = "";
    $("#conn").append(el("span", { class: "tag bad", text: "连接检查失败" }));
  }
}

async function route() {
  const hash = location.hash.replace(/^#\/?/, "");
  const [section, sub] = hash.split("/");
  renderNav(section, sub);
  const content = clear($("#content"));
  try {
    if (section === "system") await renderSystem(content);
    else if (section === "metadata") await renderMetadata(content, sub || "device");
    else if (section === "data") await renderData(content, sub || "event");
    else if (section === "mappings") await renderMappings(content);
    else if (section === "settings") await renderSettings(content);
    else await renderOverview(content);
  } catch (error) {
    content.append(el("div", { class: "panel" }, el("div", { class: "muted", text: "加载失败：" + error.message })));
    toast(error.message, "error");
  }
  refreshConnection();
}

/* ------------------------------------------------------------------ */
/* overview                                                           */
/* ------------------------------------------------------------------ */

async function renderOverview(root) {
  setTitle("控制台概览", "EdgeX Foundry");
  const data = await api("/overview");
  const cards = el("div", { class: "cards" });
  for (const item of data.counts) {
    cards.append(
      el("div", { class: "card" }, [
        el("div", { class: "value", text: item.error ? "—" : item.total.toLocaleString() }),
        el("div", { class: "label", text: item.label }),
        item.error ? el("div", { class: "error", text: item.error }) : null,
      ]),
    );
  }
  root.append(cards);
  root.append(
    el("div", { class: "panel" }, [
      el("h2", { text: "服务健康" }),
      table(["service", "status", "url", "message"], data.health || []),
    ]),
  );
}

/* ------------------------------------------------------------------ */
/* system                                                             */
/* ------------------------------------------------------------------ */

async function renderSystem(root) {
  setTitle("系统服务", "EdgeX Foundry > 系统服务");
  const [health, services] = await Promise.all([api("/health"), api("/services")]);
  root.append(
    el("div", { class: "panel" }, [
      el("h2", { text: "服务健康" }),
      table(["service", "status", "url", "message"], health.services || []),
    ]),
  );

  const registrations = services.items || [];
  root.append(
    el("div", { class: "panel" }, [
      el("h2", { text: "注册中心（core-keeper）" }),
      registrations.length
        ? table(["serviceId", "host", "port", "status"], registrations)
        : el("div", { class: "muted", text: "注册中心暂无服务。" }),
    ]),
  );

  const serviceInput = el("input", { type: "text", value: "core-metadata" });
  const actionSelect = el("select");
  for (const action of ["metrics", "config"]) actionSelect.append(el("option", { value: action, text: action }));
  const resultBox = el("div", { class: "mono muted", text: "尚未查询。" });

  async function runQuery() {
    try {
      const data = await api(`/system/${actionSelect.value}?services=${encodeURIComponent(serviceInput.value)}`);
      resultBox.textContent = JSON.stringify(data.result, null, 2);
    } catch (error) {
      resultBox.textContent = "查询失败：" + error.message;
      toast(error.message, "error");
    }
  }

  const opService = el("input", { type: "text", value: "core-metadata" });
  const opAction = el("select");
  for (const action of ["start", "stop", "restart"]) opAction.append(el("option", { value: action, text: action }));

  root.append(
    el("div", { class: "panel" }, [
      el("h2", { text: "服务配置与指标（sys-mgmt-agent）" }),
      el("div", { class: "toolbar" }, [
        el("label", { class: "field" }, [el("span", { text: "服务名称" }), serviceInput]),
        el("label", { class: "field" }, [el("span", { text: "查询类型" }), actionSelect]),
        el("button", { class: "btn primary", type: "button", text: "查询", onclick: runQuery }),
      ]),
      resultBox,
      el("h2", { text: "服务启停", style: "margin-top:18px" }),
      el("div", { class: "toolbar" }, [
        el("label", { class: "field" }, [el("span", { text: "服务名称" }), opService]),
        el("label", { class: "field" }, [el("span", { text: "操作" }), opAction]),
        el("button", {
          class: "btn danger",
          type: "button",
          text: "执行操作",
          onclick: async () => {
            try {
              const data = await api("/system/operation", {
                method: "POST",
                body: { serviceName: opService.value, action: opAction.value },
              });
              toast("操作已提交：" + JSON.stringify(data.result), "ok");
            } catch (error) {
              toast(error.message, "error");
            }
          },
        }),
      ]),
      el("div", { class: "muted", text: "提示：当前部署未运行 sys-mgmt-agent，启停/指标接口可能不可用。" }),
    ]),
  );
}

/* ------------------------------------------------------------------ */
/* metadata                                                           */
/* ------------------------------------------------------------------ */

async function renderMetadata(root, kind) {
  const label = META_LABEL[kind] || kind;
  setTitle(label, `EdgeX Foundry > 元数据 > ${label}`);
  const offsetRef = { offset: 0, limit: 20, filter: "", value: "", search: "" };
  const host = el("div");
  root.append(host);

  async function load() {
    const query = new URLSearchParams({
      offset: String(offsetRef.offset),
      limit: String(offsetRef.limit),
      filter: offsetRef.filter,
      value: offsetRef.value,
    });
    const data = await api(`/metadata/${kind}?${query.toString()}`);
    let items = data.items || [];
    if (offsetRef.search) {
      const needle = offsetRef.search.toLowerCase();
      items = items.filter((row) => JSON.stringify(row).toLowerCase().includes(needle));
    }
    clear(host);

    const search = el("input", { type: "text", value: offsetRef.search, placeholder: "筛选当前页" });
    search.addEventListener("input", () => {
      offsetRef.search = search.value;
      load().catch((error) => toast(error.message, "error"));
    });
    const filter = el("select");
    for (const option of kind === "device" ? ["", "service", "profile"] : [""]) {
      filter.append(el("option", { value: option, text: option || "全部" }));
    }
    filter.value = offsetRef.filter;
    const valueInput = el("input", { type: "text", value: offsetRef.value, placeholder: "精确名称" });
    const size = el("select");
    for (const option of [20, 50, 100]) size.append(el("option", { value: String(option), text: option + " 条/页" }));
    size.value = String(offsetRef.limit);

    const page = Math.floor(offsetRef.offset / offsetRef.limit) + 1;
    const pageInput = el("input", { type: "number", min: "1", value: String(page), style: "width:90px" });

    const toolbar = el("div", { class: "toolbar" }, [
      el("label", { class: "field" }, [el("span", { text: "筛选当前页" }), search]),
      ...(kind === "device"
        ? [el("label", { class: "field" }, [el("span", { text: "服务端筛选" }), filter])]
        : []),
      ...(kind === "device"
        ? [el("label", { class: "field" }, [el("span", { text: "精确名称" }), valueInput])]
        : []),
      el("label", { class: "field" }, [el("span", { text: "每页条数" }), size]),
      el("label", { class: "field" }, [el("span", { text: "页码" }), pageInput]),
      el("button", {
        class: "btn",
        type: "button",
        text: "应用",
        onclick: () => {
          offsetRef.limit = Number(size.value);
          offsetRef.filter = filter.value;
          offsetRef.value = valueInput.value;
          offsetRef.offset = (Number(pageInput.value) - 1) * offsetRef.limit;
          load().catch((error) => toast(error.message, "error"));
        },
      }),
      el("div", { class: "spacer" }),
      el("button", {
        class: "btn primary",
        type: "button",
        text: "＋ 新建" + label,
        onclick: () => openCreate(),
      }),
    ]);

    host.append(toolbar);
    host.append(el("div", { class: "muted", text: `共 ${data.totalCount} 条 · 当前显示 ${items.length} 条` }));
    if (!items.length) {
      host.append(el("div", { class: "panel" }, el("div", { class: "muted", text: "暂无匹配数据。" })));
      return;
    }

    const actions = (row) => {
      const buttons = [
        el("button", { class: "btn small", type: "button", text: "编辑", onclick: () => openEdit(row) }),
      ];
      if (kind === "device") {
        buttons.push(
          el("button", { class: "btn small", type: "button", text: "命令", onclick: () => openCommands(row.name) }),
        );
      }
      if (kind === "profile") {
        buttons.push(
          el("button", {
            class: "btn small",
            type: "button",
            text: "导出 YAML",
            onclick: () => window.open(`/api/profile/${encodeURIComponent(row.name)}/yaml`, "_blank"),
          }),
        );
      }
      buttons.push(
        el("button", {
          class: "btn small danger",
          type: "button",
          text: "删除",
          onclick: () => remove(row.name),
        }),
      );
      return buttons;
    };
    host.append(el("div", { class: "panel" }, table(META_COLUMNS[kind] || ["name"], items, actions)));
  }

  function openCreate() {
    const { form, inputs } = buildForm(META_FIELDS[kind] || [{ key: "name", label: "名称", type: "text" }]);
    form.append(
      el("div", { class: "actions" }, [
        el("button", {
          class: "btn primary",
          type: "button",
          text: "创建",
          onclick: async () => {
            try {
              const record = collectForm(inputs);
              await api(`/metadata/${kind}`, { method: "POST", body: record });
              closeModal();
              toast("已创建。", "ok");
              load().catch(() => {});
            } catch (error) {
              toast(error.message, "error");
            }
          },
        }),
      ]),
    );
    openModal("新建" + label, form);
  }

  function openEdit(row) {
    const { form, inputs } = buildForm(META_FIELDS[kind] || [{ key: "name", label: "名称", type: "text" }], row, kind !== "service" ? "name" : null);
    form.append(
      el("div", { class: "actions" }, [
        el("button", {
          class: "btn primary",
          type: "button",
          text: "保存",
          onclick: async () => {
            try {
              const record = collectForm(inputs, { name: row.name });
              await api(`/metadata/${kind}`, { method: "PUT", body: record });
              closeModal();
              toast("已保存。", "ok");
              load().catch(() => {});
            } catch (error) {
              toast(error.message, "error");
            }
          },
        }),
      ]),
    );
    openModal("编辑 " + row.name, form);
  }

  async function remove(name) {
    if (!window.confirm(`确认删除 ${name}？`)) return;
    try {
      await api(`/metadata/${kind}/${encodeURIComponent(name)}`, { method: "DELETE" });
      toast("已删除。", "ok");
      load().catch(() => {});
    } catch (error) {
      toast(error.message, "error");
    }
  }

  async function openCommands(device) {
    const box = el("div", { class: "mono muted", text: "加载中…" });
    openModal("设备命令 · " + device, box);
    try {
      const data = await api(`/devices/${encodeURIComponent(device)}/commands`);
      const commands = (data && data.coreCommands) || [];
      clear(box);
      if (!commands.length) {
        box.textContent = "该设备没有可执行命令。";
        return;
      }
      for (const command of commands) {
        const bodyInput = el("textarea", { rows: 3 });
        bodyInput.value = JSON.stringify({}, null, 2);
        box.append(
          el("div", { class: "panel" }, [
            el("h2", { text: command.name }),
            el("div", { class: "muted", text: "可执行：" + [command.get ? "GET" : null, command.set ? "PUT" : null].filter(Boolean).join(" / ") }),
            command.get
              ? el("button", {
                  class: "btn small",
                  type: "button",
                  text: "执行 GET",
                  onclick: async () => {
                    try {
                      const result = await api(`/devices/${encodeURIComponent(device)}/commands/${encodeURIComponent(command.name)}`);
                      toast("GET 成功", "ok");
                      openModal("命令结果", el("pre", { class: "mono", text: JSON.stringify(result, null, 2) }));
                    } catch (error) {
                      toast(error.message, "error");
                    }
                  },
                })
              : null,
            command.set
              ? el("div", {}, [
                  el("label", { class: "field" }, [el("span", { text: "PUT 参数（JSON，值为字符串）" }), bodyInput]),
                  el("button", {
                    class: "btn primary small",
                    type: "button",
                    text: "执行 PUT",
                    onclick: async () => {
                      try {
                        const result = await api(`/devices/${encodeURIComponent(device)}/commands/${encodeURIComponent(command.name)}`, {
                          method: "PUT",
                          body: JSON.parse(bodyInput.value || "{}"),
                        });
                        toast("PUT 成功", "ok");
                        openModal("命令结果", el("pre", { class: "mono", text: JSON.stringify(result, null, 2) }));
                      } catch (error) {
                        toast(error.message, "error");
                      }
                    },
                  }),
                ])
              : null,
          ]),
        );
      }
    } catch (error) {
      clear(box).textContent = "加载失败：" + error.message;
    }
  }

  await load();
}

/* ------------------------------------------------------------------ */
/* data center                                                        */
/* ------------------------------------------------------------------ */

function readingChart(records) {
  const points = records
    .map((row) => ({ value: Number(row.value), origin: Number(row.origin) }))
    .filter((point) => Number.isFinite(point.value) && Number.isFinite(point.origin));
  if (points.length < 2) return null;
  points.sort((a, b) => a.origin - b.origin);
  const width = 720;
  const height = 160;
  const values = points.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const step = width / (points.length - 1);
  const polyline = points
    .map((point, index) => `${(index * step).toFixed(1)},${(height - ((point.value - min) / span) * height).toFixed(1)}`)
    .join(" ");
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.setAttribute("width", "100%");
  svg.setAttribute("height", String(height));
  const line = document.createElementNS("http://www.w3.org/2000/svg", "polyline");
  line.setAttribute("points", polyline);
  line.setAttribute("fill", "none");
  line.setAttribute("stroke", "#168b9c");
  line.setAttribute("stroke-width", "2");
  svg.append(line);
  return el("div", { class: "panel" }, [el("h2", { text: "数值趋势" }), svg]);
}

async function renderData(root, kind) {
  const label = kind === "event" ? "事件" : "读数";
  setTitle(label, `EdgeX Foundry > 数据中心 > ${label}`);
  const state = { device: "", offset: 0, limit: 20 };
  const host = el("div");
  root.append(host);

  async function load() {
    const query = new URLSearchParams({
      device: state.device,
      offset: String(state.offset),
      limit: String(state.limit),
    });
    const data = await api(`/data/${kind}?${query.toString()}`);
    const items = data.items || [];
    clear(host);
    const deviceInput = el("input", { type: "text", value: state.device, placeholder: "设备名称" });
    host.append(
      el("div", { class: "toolbar" }, [
        el("label", { class: "field" }, [el("span", { text: "设备名称（服务端精确查询）" }), deviceInput]),
        el("button", {
          class: "btn",
          type: "button",
          text: "查询",
          onclick: () => {
            state.device = deviceInput.value;
            state.offset = 0;
            load().catch((error) => toast(error.message, "error"));
          },
        }),
        el("button", {
          class: "btn",
          type: "button",
          text: "下一页",
          onclick: () => {
            state.offset += state.limit;
            load().catch((error) => toast(error.message, "error"));
          },
        }),
      ]),
    );
    host.append(el("div", { class: "muted", text: `共 ${data.totalCount} 条 · 当前显示 ${items.length} 条` }));
    const columns = kind === "event"
      ? ["id", "deviceName", "profileName", "sourceName", "origin"]
      : ["id", "deviceName", "resourceName", "valueType", "value", "origin"];
    host.append(el("div", { class: "panel" }, table(columns, items)));
    if (kind === "reading") {
      const chart = readingChart(items);
      if (chart) host.append(chart);
    }
  }

  await load();
}

/* ------------------------------------------------------------------ */
/* mappings                                                           */
/* ------------------------------------------------------------------ */

const MAPPING_FILTERS = ["all", "north/device", "north/profile", "south/device", "south/profile"];

function mappingToForm(mapping) {
  const record = { metadata: mapping.metadata || {} };
  for (const side of ["north", "south"]) {
    const point = mapping[side + "Point"] || {};
    record[side + "DeviceName"] = point[side + "DeviceName"] || "";
    record[side + "ProfileName"] = point[side + "ProfileName"] || "";
    record[side + "ResourceName"] = point[side + "ResourceName"] || "";
  }
  return record;
}

function formToMapping(record, id) {
  const side = (name) => ({
    [name + "DeviceName"]: record[name + "DeviceName"],
    [name + "ProfileName"]: record[name + "ProfileName"],
    [name + "ResourceName"]: record[name + "ResourceName"],
  });
  const body = { northPoint: side("north"), southPoint: side("south"), metadata: record.metadata || {} };
  if (id) body.id = id;
  return body;
}

async function renderMappings(root) {
  setTitle("映射管理", "EdgeX Foundry > 映射管理");
  const state = { filter: "all", name: "", offset: 0, limit: 20 };
  const host = el("div");
  root.append(host);

  async function load() {
    const query = new URLSearchParams({
      filter: state.filter,
      name: state.name,
      offset: String(state.offset),
      limit: String(state.limit),
    });
    const data = await api(`/mappings?${query.toString()}`);
    const items = data.items || [];
    clear(host);

    const filterSelect = el("select");
    for (const option of MAPPING_FILTERS) filterSelect.append(el("option", { value: option, text: option }));
    filterSelect.value = state.filter;
    const nameInput = el("input", { type: "text", value: state.name, placeholder: "设备 / 模型名称" });

    host.append(
      el("div", { class: "toolbar" }, [
        el("label", { class: "field" }, [el("span", { text: "筛选范围" }), filterSelect]),
        el("label", { class: "field" }, [el("span", { text: "名称" }), nameInput]),
        el("button", {
          class: "btn",
          type: "button",
          text: "查询",
          onclick: () => {
            const nextFilter = filterSelect.value;
            const nextName = nameInput.value.trim();
            if (nextFilter !== "all" && !nextName) {
              toast("请填写需要精确筛选的设备或模型名称", "error");
              return;
            }
            state.filter = nextFilter;
            state.name = nextName;
            state.offset = 0;
            load().catch((error) => toast(error.message, "error"));
          },
        }),
        el("div", { class: "spacer" }),
        el("button", { class: "btn primary", type: "button", text: "＋ 新增映射", onclick: () => openCreate() }),
        el("button", { class: "btn", type: "button", text: "查询映射", onclick: () => openLookup() }),
      ]),
    );
    host.append(el("div", { class: "muted", text: `共 ${data.totalCount} 条 · 当前显示 ${items.length} 条` }));
    if (!items.length) {
      host.append(el("div", { class: "panel" }, el("div", { class: "muted", text: "暂无映射。" })));
      return;
    }
    const actions = (row) => [
      el("button", { class: "btn small", type: "button", text: "编辑", onclick: () => openEdit(row) }),
      el("button", { class: "btn small danger", type: "button", text: "删除", onclick: () => remove(row.id) }),
    ];
    host.append(
      el("div", { class: "panel" }, table(["id", "northPoint", "southPoint", "metadata", "created", "modified"], items, actions)),
    );
  }

  function openCreate() {
    const { form, inputs } = buildForm(MAPPING_FIELDS);
    form.append(
      el("div", { class: "actions" }, [
        el("button", {
          class: "btn primary",
          type: "button",
          text: "新增映射",
          onclick: async () => {
            try {
              await api("/mappings", { method: "POST", body: formToMapping(collectForm(inputs)) });
              closeModal();
              toast("映射已创建。", "ok");
              load().catch(() => {});
            } catch (error) {
              toast(error.message, "error");
            }
          },
        }),
      ]),
    );
    openModal("新增映射", form);
  }

  function openEdit(row) {
    const { form, inputs } = buildForm(MAPPING_FIELDS, mappingToForm(row));
    form.append(
      el("div", { class: "actions" }, [
        el("button", {
          class: "btn primary",
          type: "button",
          text: "保存修改",
          onclick: async () => {
            try {
              await api("/mappings", { method: "PUT", body: formToMapping(collectForm(inputs), row.id) });
              closeModal();
              toast("映射已更新。", "ok");
              load().catch(() => {});
            } catch (error) {
              toast(error.message, "error");
            }
          },
        }),
      ]),
    );
    openModal("编辑映射 " + row.id, form);
  }

  function openLookup() {
    const direction = el("select");
    for (const option of ["forward", "reverse"]) {
      direction.append(el("option", { value: option, text: option === "forward" ? "北向 → 南向" : "南向 → 北向" }));
    }
    const device = el("input", { type: "text" });
    const profile = el("input", { type: "text" });
    const resource = el("input", { type: "text" });
    const result = el("div", { class: "mono muted", text: "尚未查询。" });
    openModal(
      "查询映射",
      el("div", {}, [
        el("label", { class: "field" }, [el("span", { text: "方向" }), direction]),
        el("label", { class: "field" }, [el("span", { text: "设备名称" }), device]),
        el("label", { class: "field" }, [el("span", { text: "模型名称" }), profile]),
        el("label", { class: "field" }, [el("span", { text: "资源名称" }), resource]),
        el("button", {
          class: "btn primary",
          type: "button",
          text: "查询",
          onclick: async () => {
            if (!device.value.trim() || !profile.value.trim() || !resource.value.trim()) {
              result.textContent = "请填写设备、模型和资源名称。";
              return;
            }
            try {
              const query = new URLSearchParams({
                direction: direction.value,
                device: device.value,
                profile: profile.value,
                resource: resource.value,
              });
              const data = await api(`/mappings/lookup?${query.toString()}`);
              result.textContent = JSON.stringify(data.mapping, null, 2);
            } catch (error) {
              result.textContent = "查询失败：" + error.message;
            }
          },
        }),
        el("h2", { text: "结果", style: "margin-top:16px" }),
        result,
      ]),
    );
  }

  async function remove(id) {
    if (!window.confirm(`确认删除映射 ${id}？`)) return;
    try {
      await api(`/mappings/${encodeURIComponent(id)}`, { method: "DELETE" });
      toast("已删除。", "ok");
      load().catch(() => {});
    } catch (error) {
      toast(error.message, "error");
    }
  }

  await load();
}

/* ------------------------------------------------------------------ */
/* settings                                                           */
/* ------------------------------------------------------------------ */

async function renderSettings(root) {
  setTitle("连接设置", "EdgeX Foundry > 连接设置");
  const config = await api("/config");
  const fields = [
    ["metadataUrl", "core-metadata 地址"],
    ["dataUrl", "core-data 地址"],
    ["commandUrl", "core-command 地址"],
    ["mappingsUrl", "support-mappings 地址"],
    ["keeperUrl", "core-keeper 地址"],
    ["systemUrl", "sys-mgmt-agent 地址（可选）"],
    ["token", "Bearer Token（可选）"],
  ];
  const inputs = {};
  const form = el("form", { class: "form" });
  for (const [key, label] of fields) {
    const input = el("input", { type: key === "token" ? "password" : "text", value: config[key] || "" });
    inputs[key] = input;
    form.append(el("label", { class: "field" }, [el("span", { text: label }), input]));
  }
  const timeout = el("input", { type: "number", min: "1", max: "60", value: String(config.timeoutSec || 8) });
  form.append(el("label", { class: "field" }, [el("span", { text: "请求超时（秒）" }), timeout]));

  const healthBox = el("div", { class: "mono muted", text: "尚未测试。" });

  form.append(
    el("div", { class: "actions" }, [
      el("button", {
        class: "btn primary",
        type: "button",
        text: "保存连接",
        onclick: async () => {
          try {
            const body = { timeoutSec: Number(timeout.value) };
            for (const [key, input] of Object.entries(inputs)) {
              if (key === "token" && input.value === "") continue;
              body[key] = input.value;
            }
            await api("/config", { method: "PUT", body });
            toast("连接配置已保存。", "ok");
            refreshConnection();
          } catch (error) {
            toast(error.message, "error");
          }
        },
      }),
      el("button", {
        class: "btn",
        type: "button",
        text: "测试连接",
        onclick: async () => {
          try {
            const data = await api("/health");
            healthBox.textContent = (data.services || [])
              .map((row) => `${row.service}: ${row.status}${row.message ? " — " + row.message : ""}`)
              .join("\n");
          } catch (error) {
            healthBox.textContent = "测试失败：" + error.message;
          }
        },
      }),
    ]),
  );

  root.append(
    el("div", { class: "panel" }, [
      el("h2", { text: "EdgeX 微服务直连地址" }),
      el("div", { class: "muted", text: "控制台直接访问各微服务 REST API，不再经过 edgex-ui-go。" }),
      form,
      el("h2", { text: "连接测试", style: "margin-top:16px" }),
      healthBox,
    ]),
  );
}

/* ------------------------------------------------------------------ */
/* boot                                                               */
/* ------------------------------------------------------------------ */

$("#refresh").addEventListener("click", () => route());
window.addEventListener("hashchange", route);
if (!location.hash) location.hash = "#/overview";
route();
