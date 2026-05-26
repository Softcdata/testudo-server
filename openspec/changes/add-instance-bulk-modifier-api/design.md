# 设计：实例级批量资源修改 API

## 1. 设计目标
1. 让用户从实例层直接表达“整实例批量替换值 / 批量删除 key”。
2. 在 server 层把高层动作展开为现有规则引擎可执行的快照。
3. 保留用户手写 `modifierRules` 作为精确覆盖手段。

## 2. 非目标
1. 不再引入模板目录或模板 CRUD。
2. 不在 Phase 1 支持正则替换、子串替换。
3. 不在 Phase 1 支持列表元素级删除。
4. 不在 server 中生成旧 reversible legacy 结构。
5. 不在 Phase 1 支持 `dataSync` 范围的 bulk action。

## 3. 核心选择

### 3.1 采用“实例动作输入 + 执行快照输出”
推荐模型：

- `bulkModifierActions`：产品层输入
- `modifierRules`：用户手写精确规则
- `modifierRuleSnapshot`：server 解析后的最终执行快照
- `modifierRuleSnapshotHash`：快照哈希

server 负责把前两者解析成第三者，operator 只消费快照。

推荐逻辑模型：

```yaml
restorePolicy:
  bulkModifierActions:
    - id: replace-db-ip
      action: replaceExactValue
      sourceValue: 10.10.0.12
      targetValue: 10.20.0.12
      applyTo: ["resourceSync"]
      directionPolicy: Auto
      enabled: true
    - id: remove-site-role
      action: removeKey
      key: testudo.softcdata.com/site-role
      applyTo: ["resourceSync"]
      directionPolicy: ForwardOnly
      enabled: true
  modifierRules:
    - id: deployment-replicas-override
      mode: reversible
      pair:
        path: /spec/replicas
        sourceValue: "2"
        targetValue: "1"
  modifierRuleSnapshot:
    - id: bulk-replace-db-ip-0001
      mode: reversible
      priority: -100
      ...
    - id: bulk-remove-site-role-0001
      mode: veleroNative
      priority: -100
      ...
    - id: deployment-replicas-override
      mode: reversible
      priority: 0
      ...
  modifierRuleSnapshotHash: sha256:abcd1234
```

字段默认值：

- `enabled=nil` 视为 `true`
- `applyTo=[]` 默认 `["resourceSync"]`
- `replaceExactValue.directionPolicy=""` 默认 `Auto`
- `removeKey.directionPolicy=""` 默认 `ForwardOnly`
- `removeKey` 展开后的每条 snapshot 规则都必须显式带上 `ForwardOnly`，不能依赖下游默认值

执行时进一步定义：

- `effectiveBulkModifierActions = bulkModifierActions` 中 `enabled != false` 的条目
- `enabled=false` 的条目只保留在产品层输入，不参与 live 扫描、snapshot 生成、0-hit 校验、protected-path 校验和摘要统计

### 3.2 Phase 1 动作只做两件事

#### `replaceExactValue`
- 扫描实例保护范围内的字符串叶子节点
- 仅当节点值与 `sourceValue` 精确相等时命中
- 每个命中点展开为一条 `reversible pair-only` 规则：
  - `pair.path=<命中路径>`
  - `pair.sourceValue=<sourceValue>`
  - `pair.targetValue=<targetValue>`
  - `priority=-100`
  - `onConflict=Fail`
  - `applyTo=<动作 applyTo>`
  - `directionPolicy=<动作 directionPolicy>`

这个动作最适合：

- IP 替换
- 域名替换
- 连接串替换
- 固定字符串值批量切换

#### `removeKey`
- 扫描实例保护范围内的对象 / map 成员键
- 仅当键名与声明的 `key` 精确相等时命中
- 每个命中点展开为一条 `veleroNative remove patch`
- 生成规则同样固定：
  - `priority=-100`
  - `onConflict=Fail`
  - `applyTo=<动作 applyTo>`
  - `directionPolicy=<动作 directionPolicy>`

这个动作 Phase 1 只处理 map/object key，不处理列表元素删除。

### 3.3 合并顺序固定为“批量生成在前，手写规则在后”
执行快照的组装顺序：

1. 先过滤出 `effectiveBulkModifierActions`
2. 按声明顺序展开 `effectiveBulkModifierActions`
3. 再把用户手写 `modifierRules` 追加到 `modifierRuleSnapshot`

这样：

- 大部分改动可以用批量动作覆盖
- 少量特殊路径仍可用手写规则做精确覆盖

但真正保证“手写规则覆盖 bulk”的关键不是数组顺序，而是优先级：

- bulk 生成规则统一写 `priority=-100`
- 手写规则保留当前行为，默认优先级仍为 `0`
- 如果用户显式把手写规则调到 `<=-100`，则用户自行承担不覆盖 batch 规则的结果

### 3.4 扫描范围固定继承实例保护范围
server 不做“扫全集群”，而是按实例当前有效保护范围扫描。

同时，snapshot 重算必须只由 bulk 相关输入触发，不得由无关字段更新触发。

create / update 的有效范围计算顺序：

1. 先拿到实例有效 spec
   - create：直接用请求体
   - update：以已有实例为基底，再叠加请求中的变更字段
   - 若本次更新未触及 bulk 相关输入，则直接保留既有 `modifierRuleSnapshot` 与 `modifierRuleSnapshotHash`
   - 若触及 bulk 相关输入后 `effectiveBulkModifierActions` 为空，则直接清理 `modifierRuleSnapshot` 与 `modifierRuleSnapshotHash`
2. 固定使用 `config.spec.sourceCluster` 作为 live 扫描集群
3. 计算有效命名空间：
   - 从 `instance.spec.namespaces` 出发
   - 若 `restorePolicy.resourceSelection.includedNamespaces` 非空，则做交集
   - 再减去 `excludedNamespaces`
4. 计算有效 label selector：
   - 若 `restorePolicy.resourceSelection.labelSelector` 非空，优先使用它
   - 否则使用 `instance.spec.labelSelector`
5. 计算候选资源类型：
   - 以 `resourceSync/drill` 的 restore 选择语义为准
   - 复用 `restorePolicy.resourceSelection` 的 include/exclude 规则
   - 若未显式声明 include 列表，则扫描发现到的 namespace-scoped restorable resources
   - 跳过 subresource、无 `list` verb 的资源
   - cluster-scoped 资源只有在 `resourceSelection` 明确允许时才进入扫描
6. 按 `groupResource + namespace + selector` 拉取 live 资源对象

说明：

- 这一步复用现有 live validation 的 `dynamic client + rest mapper` 能力
- 不是在 server 内再造第三套资源定位框架

bulk 相关输入至少包括：

- `restorePolicy.bulkModifierActions`
- `restorePolicy.modifierRules`
- `restorePolicy.resourceSelection`
- `spec.namespaces`
- `spec.labelSelector`
- `spec.config`

像 `description`、`podRestoreMethod`、`skipPodReadyCheck` 这类无关字段更新，不得重算 snapshot。

### 3.5 展开算法必须稳定且可复现
每个 action 的命中结果必须先做稳定排序，再生成规则 ID 和 snapshot。

稳定排序键建议固定为：

1. `groupResource`
2. `namespace`
3. `resourceName`
4. `jsonPointerPath`

然后按排序后的顺序生成：

- `bulk-<action-id>-0001`
- `bulk-<action-id>-0002`
- ...

这样：

- 相同 live 资源输入下，snapshot 顺序稳定
- `modifierRuleSnapshotHash` 可以稳定复现

### 3.6 最终快照必须复用现有校验器
bulk action 展开完成后，不直接写实例，必须先走已有规则校验链：

1. 组装一个临时 `RestorePolicy`
2. 将 `modifierRuleSnapshot` 填入 `ModifierRules`
3. 复用：
   - `validateRestorePolicyModifierRules(...)`
   - `validateRestorePolicyModifierRulesLive(...)`

这样可以直接继承：

- protected-path 校验
- pair-only 校验
- metadata string 语义校验
- live 路径定位校验
- conflict / zero-match 之外的现有治理限制

### 3.7 清空 bulk 输入必须清理执行快照
当用户把 `bulkModifierActions` 清空、显式移除，或全部改成 `enabled=false` 时，server 必须：

1. 清理 `modifierRuleSnapshot`
2. 清理 `modifierRuleSnapshotHash`
3. 保留用户手写 `modifierRules`

原因：

- operator 会把 snapshot 视为更高优先级输入
- 如果不清理，旧 bulk 规则会在用户删除 bulk action 后继续执行

### 3.8 失败关闭而不是静默 no-op
以下情况必须拒绝实例写入：

- 已启用动作命中 0 个具体修改点
- 已启用动作试图修改受保护路径
- 已启用 `removeKey` 命中的是不支持的结构
- 已启用动作展开后规则不符合当前正式 contract
- 同一路径被不同已启用 bulk action 展开成不同值
- `applyTo` 包含 `dataSync`

原因：

- 用户使用的是高层批量入口，不能让它看起来“配置成功”但实际没命中任何对象。
- 已禁用动作只是暂存配置，不应该触发展开期失败。

## 4. 扫描边界
Phase 1 建议：

- 只扫描实例保护范围内、server 已能定位的 live 资源
- 继续继承现有 protected-path 约束
- 对批量扫描再附加一层安全限制：
  - 只处理字符串叶子节点与对象 / map 成员键
- 不碰 `/status`
- 不碰 `metadata.finalizers`
- 不碰 `metadata.ownerReferences`
- `removeKey` 只处理 object/map key，不处理数组元素
- `replaceExactValue` 只处理字符串叶子节点，不处理数字、布尔、对象、数组

后续若要支持更多结构，单独起 proposal。

## 5. hash 生成策略
`modifierRuleSnapshotHash` 必须只覆盖最终 `modifierRuleSnapshot`，不覆盖 `bulkModifierActions` 原始输入。

推荐算法：

1. 先生成排序稳定的 `modifierRuleSnapshot`
2. 对 snapshot 做标准 JSON 序列化
3. 计算 `sha256`
4. 存储为 `sha256:<hex>`

这样 operator 与排障链路看到的 hash 就对应“最终执行规则”，而不是上层动作文本。

## 6. 风险
1. “整实例扫描”如果边界太宽，容易误中；Phase 1 必须坚持 exact match。
2. 一个动作可能展开出很多规则，需要配合 snapshot hash 与摘要排障。
3. 如果 discovery 对部分 API 组返回不稳定结果，需要在实现里跳过 subresource 和无 `list` verb 的资源，并对显式 include 的失败资源明确报错。
4. 如果实例资源频繁变化，bulk action 的展开结果会随提交时 live 资源而变化，这是绑定时快照的预期行为。

## 7. 验证策略
1. handler test：请求体读写与字段回显。
2. 展开单测：`replaceExactValue` 生成 pair-only。
3. 展开单测：`removeKey` 生成 veleroNative remove。
4. 展开单测：`removeKey` 生成的 snapshot 明确写入 `directionPolicy=ForwardOnly`。
5. 展开单测：snapshot 排序与 hash 稳定。
6. 回归测试：与手写 `modifierRules` 混用时手写默认优先级可覆盖 bulk。
7. 回归测试：无关字段更新不会重算 snapshot。
8. 回归测试：清空或全部禁用 `bulkModifierActions` 时会同步清理 snapshot/hash。
9. 回归测试：`enabled=false` 的动作不会触发 0 命中、受保护路径、结构不支持这类展开失败。
10. 回归测试：0 命中、受保护路径、结构不支持、`dataSync` 输入时失败关闭。
