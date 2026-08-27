-- Generates test/testdata/pairs_orders.txt: the shipped luajit's pairs()
-- iteration order over int-keyed tables (keys 0..37, up to 4 distinct),
-- the domain tree/luatable.go emulates for the GV additions merge.
-- Run from tools/: luajit gen_pairs_orders.lua ../test/testdata/pairs_orders.txt
local function order(seq)
  local t = {}
  for _, k in ipairs(seq) do t[k] = true end
  local out = {}
  for k in pairs(t) do out[#out+1] = k end
  return table.concat(out, ",")
end
local out = io.open(arg[1], "wb")
-- all sequences of distinct keys, len 1..3, keys 0..37
for a = 0, 37 do
  out:write(a, "|", order({a}), "\n")
  for b = 0, 37 do if b ~= a then
    out:write(a, " ", b, "|", order({a, b}), "\n")
    for c = 0, 37 do if c ~= a and c ~= b then
      out:write(a, " ", b, " ", c, "|", order({a, b, c}), "\n")
    end end
  end end
end
-- deterministic pseudo-random len-4 sequences
local state = 12345
local function rnd(n) state = (state * 1103515245 + 12345) % 2147483648; return state % n end
for i = 1, 200000 do
  local seen, seq = {}, {}
  while #seq < 4 do
    local k = rnd(38)
    if not seen[k] then seen[k] = true; seq[#seq+1] = k end
  end
  out:write(table.concat(seq, " "), "|", order(seq), "\n")
end
out:close()
