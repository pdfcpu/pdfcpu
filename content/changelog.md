---
layout: default
title: "Changelog"
---

# Changelog

Entries link to GitHub releases or commits where available.

<style>
.changelog-meta {
  align-items: baseline;
  column-gap: 12px;
  font-size: 1.25rem;
  font-weight: 700;
  line-height: 1.25;
  margin-bottom: 10px;
}

.changelog-meta a {
  font-size: 0.85rem;
  font-weight: 800;
}

.changelog-meta time,
.changelog-kind {
  font-size: 0.85rem;
}
</style>

<div class="changelog-list">

<h2 class="changelog-year">2026</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2026-08-11">2026-08-11</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.15.0">v0.15.0</a>
  </div>
  <p>
  Improve large-corpus validation diagnostics by streaming per-file failures, reporting a compact summary, and adding
  <code>validate --progress</code> for quiet runs.<br>
  Harden malformed PDF reading, validation, optimization, encryption handling, and signature diagnostics.<br>
  Add automatic CJK wrapping for text watermarks.<br>
  Fix #1427, #1459.
  </p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2026-08-03">2026-08-03</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.14.0">v0.14.0</a>
  </div>
  <p>
  Harden API and CLI error handling.<br>
  Harden signature, timestamp, PKCS#7, certificate-path, and revocation processing.<br>
  Eliminate dependencies by using standard-library error handling and internal LZW and PKCS#7
  implementations.<br><br>
  Fix #415, #866, #990, #1051, #1088, #1091, #1101, #1123, #1127, #1161, #1265, #1274, #1279, #1282, #1289,
  #1302, #1311, #1325, #1326, #1340, #1383, #1385, #1387, #1403, #1404, #1417, #1419, #1431, #1437-#1440,
  #1448. Add regression coverage for #399, #401, #933, #1059, and #1271.
  </p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2026-06-09">2026-06-09</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.13.0">v0.13.0</a>
  </div>
  <p>CLI: Add stdin/stdout support and --force.<br> 
  Refactor command plumbing and parameter handling. <br>
  Harden stream parsing, filter decoding, file path handling, and parser limits. <br>
  Reduce default binary size by moving bundled EUTL trust-list certificates behind the pdfcpu_eutl build tag.<br> 
  Fix #513, #801, #1291, #1296, #1316, #1317, #1327, #1359, #1364, #1373, #1375, #1393, #1394, #1396, #1402, #1410, #1411. </p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2026-05-11">2026-05-11</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.12.1">v0.12.1</a>
  </div>
  <p>Fix #1319, #1322, #1357, #1381, #1388, #1389</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2026-04-22">2026-04-22</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.12.0">v0.12.0</a>
  </div>
  <p>Migrate cli to cobra.<br>Fix #399, #642, #1055, #1201, #1211, #1215, #1229, #1231, #1255, #1261, #1263, #1267, #1268, #1276, #1278, #1280, #1285, #1292, #1297-#1299, #1307, #1329-#1331, #1334, #1341, #1345, #1353, #1382</p>
</article>


<h2 class="changelog-year">2025</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2025-10-21">2025-10-21</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.11.1">v0.11.1</a>
  </div>
  <p>Fix #846, #1097, #1112, #1156, #1166, #1173, #1176, #1177, #1183, #1185, #1187, #1188, #1189, #1193-#1195, #1202, #1203, #1216, #1226, #1230, #1231, #1235</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2025-05-28">2025-05-28</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.11.0">v0.11.0</a>
  </div>
  <p>Add cert inspect command.<br> Fix #1056, #1085, #1107, #1113, #1117-#1119, #1142, #1149, #1152, #1163, #1165, #1168, #1171</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2025-04-23">2025-04-23</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.10.2">v0.10.2</a>
  </div>
  <p>Add signature &amp; cert commands.<br> Fix #888, #972, #973, #984, #985, #987, #988, #991, #999, #1001, #1007, #1008, #1010, #1013, #1015-#1017, #1019, #1021, #1025, #1027, #1029, #1034, #1036, #1041, #1047, #1049, #1058, #1064, #1065, #1066, #1067, #1072, #1073, #1076, #1077, #1080, #1081, #1089, #1090, #1098, #1099, #1100, #1111, #1114, #1116</p>
</article>


<h2 class="changelog-year">2024</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2024-10-24">2024-10-24</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.9.1">v0.9.1</a>
  </div>
  <p>Fix config file handling</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2024-10-24">2024-10-24</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.9.0">v0.9.0</a>
  </div>
  <p>Add images list, extract, update cmds.<br> Add config list, reset cmds. <br> Add offline flag.<br> Fix #455, #859, #868, #897, #935, #940, #941, #947, #948, #953, #951, #953, #955, #961, #965</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2024-08-31">2024-08-31</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.8.1">v0.8.1</a>
  </div>
  <p>Improve CJK, annotation support.<br> Fix #628, #687, #767, #819, #830, #862, #867, #871, #881, #884-#887, #890, #891, #895, #898, #903, #907, #908, #910-#912, #914, #915, #918, #921, #924, #926, #930-#932</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2024-04-25">2024-04-25</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.8.0">v0.8.0</a>
  </div>
  <p>PDF 2.0 encryption, parser speedup, booklet enhancements.<br> Fix #821, #823, #826, #828, #832, #834, #835, #838, #839, #841, #844, #849, #851, #852, #855</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2024-03-04">2024-03-04</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.7.0">v0.7.0</a>
  </div>
  <p>Add zoom command, basic PDF 2.0 updating.<br> Fix #628, #724, #756, #758-#760, #765-#766, #769-#774, #780, #783-#784, #786-#787, #793-#796, #798, #802, #805-#811, #813-815, #818</p>
</article>


<h2 class="changelog-year">2023</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2023-12-10">2023-12-10</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.6.0">v0.6.0</a>
  </div>
  <p>Add pagelayout, pagemode, viewerpref cmds, basic PDF 2.0 validation.<br> Fix #373, #472, #473, #635, #665, #689, #701, #705, #706, #708, #710, #711, #713, #716, #717, #722, #723, #727, #731-733, #734, #736-740, #742,747</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2023-08-20">2023-08-20</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.5.0">v0.5.0</a>
  </div>
  <p>Add bookmarks command.<br> Fix #506, #604, #621, #657, #659, #660, #663, #664, #666, #667, #669, #671</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2023-07-20">2023-07-20</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.4.2">v0.4.2</a>
  </div>
  <p>Bookmark support for merging.<br> Fix #606, #608, #617, #618, #622, #623, #624, #626, #627, #630-#632, #635-#637, #644, #649, #650, #654</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2023-05-06">2023-05-06</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.4.1">v0.4.1</a>
  </div>
  <p>Add cut, ndown, poster commands</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2023-02-28">2023-02-28</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.4.0">v0.4.0</a>
  </div>
  <p>Add form, resize commands</p>
</article>


<h2 class="changelog-year">2021</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-12-04">2021-12-04</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/e9f927d44d0f2a8bbf7413692595f4f047f6371c">e9f927d</a>
  </div>
  <p>Fix #396, add config command</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-12-01">2021-12-01</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/a8a031e36b4c08f7dfc63b8d34156263468e9bd5">a8a031e</a>
  </div>
  <p>Fix #398</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-11-30">2021-11-30</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.13">v0.3.13</a>
  </div>
  <p>Add create command.<br> Fix 335, #349, #353, #354, #356, #358, #362, #366, #371, #380, #381, #386, #387, #394, #388</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-07-12">2021-07-12</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.12">v0.3.12</a>
  </div>
  <p>Add annotations, images commands.<br> Fix #300, #302, #323, #324, #329, #331-336, #338, #341-343, #347, #350</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-04-05">2021-04-05</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.11">v0.3.11</a>
  </div>
  <p>Add right to left stamping</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-04-05">2021-04-05</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.10">v0.3.10</a>
  </div>
  <p>Support webp, RTL Unicode Text.<br> Fix #271, #273, #285, #287, #293-#299, #301, #303, #305, #307, #311, #313, #316, #319</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2021-02-13">2021-02-13</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.9">v0.3.9</a>
  </div>
  <p>Add booklet cmd.<br> Fix #276, #279, #280, #285, #288, #290, #291</p>
</article>


<h2 class="changelog-year">2020</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-12-24">2020-12-24</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.8">v0.3.8</a>
  </div>
  <p>Add boxes, crop commands.<br> Fix #210, #216, #236, #238, #241, #244, #245, #250, #252, #256, #258, #259, #262, #264, #265, #268</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-11-04">2020-11-04</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.7">v0.3.7</a>
  </div>
  <p>Add CJKV font support.<br> Fix #233</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-09-30">2020-09-30</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.6">v0.3.6</a>
  </div>
  <p>Fix #218, #220-#224, #231,#232. Add config dir &amp; file</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-08-31">2020-08-31</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.5">v0.3.5</a>
  </div>
  <p>Fix #145, #207, #208, #213, #215</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-06-28">2020-06-28</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.4">v0.3.4</a>
  </div>
  <p>Fix #100,#102,#177,#180,#187. Fix #191-197, #199-202</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-05-27">2020-05-27</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/58a1e278d00ee588f6e00b5b74f1fa965c5ce889">58a1e27</a>
  </div>
  <p>Fix #174</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-05-26">2020-05-26</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/ed59dc35b4f5f7051141d5c11d4e887d453daa58">ed59dc3</a>
  </div>
  <p>Fix #192</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-05-25">2020-05-25</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.3">v0.3.3</a>
  </div>
  <p>stamps: Add hAlign, margins, border.<br> Fix #157,#170,#173,#175,#181-184,#188</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2020-01-04">2020-01-04</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.2">v0.3.2</a>
  </div>
  <p>Support multi-stamping.<br> Add keywords, properties commands.<br> Add collect, portfolio commands.<br> Fix #112,#140,#143,#144,#146,#148,#152</p>
</article>


<h2 class="changelog-year">2019</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-11-15">2019-11-15</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3.1">v0.3.1</a>
  </div>
  <p>TrueType support.<br> Fix #126,133,137,138</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-11-15">2019-11-15</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.3">v0.3</a>
  </div>
  <p>Fix #113,#114,#117,#119,#121,#123,#130</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-09-23">2019-09-23</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2.5">v0.2.5</a>
  </div>
  <p>Fix #101, #103, #107-#109</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-08-28">2019-08-28</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2.4">v0.2.4</a>
  </div>
  <p>Fix #100, #104. Use x/image/ccitt.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-08-11">2019-08-11</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2.3">v0.2.3</a>
  </div>
  <p>Transfer repo to org</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-08-01">2019-08-01</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2.2">v0.2.2</a>
  </div>
  <p>Fix #95-#97</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-07-22">2019-07-22</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/18994fd6776f425631cc195b28db99fd91d7c76f">18994fd</a>
  </div>
  <p>Fix #94</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-07-15">2019-07-15</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2.1">v0.2.1</a>
  </div>
  <p>Fix #92, #93</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-07-14">2019-07-14</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.2">v0.2</a>
  </div>
  <p>Redesign API, info cmd.<br> Fix #87,#89-#91</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-06-17">2019-06-17</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.25">v0.1.25</a>
  </div>
  <p>Fix #88</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-06-16">2019-06-16</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.24">v0.1.24</a>
  </div>
  <p>Add AES-256 encryption</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-06-10">2019-06-10</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/86b344515ab4f12e80ea26f3c0e519e28047274f">86b3445</a>
  </div>
  <p>Fix #82, #86 repairs corrupt xref sections.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-06-03">2019-06-03</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/33da2ddb3686ab98567f1abefbc62e8d15ed2720">33da2dd</a>
  </div>
  <p>Fix #85</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-05-10">2019-05-10</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/f9a4092442562289304485a9d0867be7e29a8084">f9a4092</a>
  </div>
  <p>Fix #80, #81</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-04-19">2019-04-19</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/9d476ddd92a1ed83f384f8da076ef8b5d242dc3c">9d476dd</a>
  </div>
  <p>Fix #77</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-04-13">2019-04-13</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/14e74ba2c2ebe2ade2aa4c8506c5e9cec2a5fbd8">14e74ba</a>
  </div>
  <p>Fix #75, #76</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-04-04">2019-04-04</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/40f60a0a25c5359e3d44c06138404079ae272622">40f60a0</a>
  </div>
  <p>Fix #74</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-30">2019-03-30</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.23">v0.1.23</a>
  </div>
  <p>Support multiline watermarks.<br> fix #27, #61, #63</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-29">2019-03-29</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/eca8f35ebe5f99da5862212e050824680f23016f">eca8f35</a>
  </div>
  <p>Fix #73</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-28">2019-03-28</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/80aa62c9dd3076f631a9f903cb7c9779d40fd3db">80aa62c</a>
  </div>
  <p>Fix #71, #72</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-24">2019-03-24</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.22">v0.1.22</a>
  </div>
  <p>Insert &amp; Remove Pages, go mod support.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-13">2019-03-13</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/967c250e0cdf31441df1c79562a00a8df3ab4a52">967c250</a>
  </div>
  <p>Fix #69: Correct name parsing.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-03-03">2019-03-03</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/067897de8ea2ff18ef6a86bf7b2da43f264c0991">067897d</a>
  </div>
  <p>Cleanup encryption.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-02-24">2019-02-24</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/6e1af9ed3b76f0306a469ed50e64e737c9f752f7">6e1af9e</a>
  </div>
  <p>Fix stamp transform calc.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-02-02">2019-02-02</time>
    <span class="changelog-kind">Commit</span>
    <a href="https://github.com/pdfcpu/pdfcpu/commit/769b2e488b07ebcc0cd4f33c651bed67d03db84e">769b2e4</a>
  </div>
  <p>Fix #64: locating lastxref.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2019-01-13">2019-01-13</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.21">v0.1.21</a>
  </div>
  <p>Add N-Up, Grid commands.<br> Fix #51, #58.</p>
</article>


<h2 class="changelog-year">2018</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-12-23">2018-12-23</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.20">v0.1.20</a>
  </div>
  <p>Add Import and Rotate commands.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-12-09">2018-12-09</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.19">v0.1.19</a>
  </div>
  <p>Add JPEG support.<br> Fix #52,#53,#54,#56.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-11-14">2018-11-14</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.18">v0.1.18</a>
  </div>
  <p>Add ReadSeeker support.<br> Fix #5,#39,#44.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-10-26">2018-10-26</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.17">v0.1.17</a>
  </div>
  <p>TIFF: Add support for CCITT decoding.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-10-21">2018-10-21</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.16">v0.1.16</a>
  </div>
  <p>CCITT fax decoding.<br> Fix #38, #40, #41.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-09-16">2018-09-16</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/0.1.15">v0.1.15</a>
  </div>
  <p>Add Stamp cmd, fork <code>x/image/tiff</code>.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-06-09">2018-06-09</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.14">v0.1.14</a>
  </div>
  <p>Extract: Write Flate as PNG.<br> Fix #25.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-05-27">2018-05-27</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.13">v0.1.13</a>
  </div>
  <p>Add Runlength filter support.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-05-22">2018-05-22</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.12">v0.1.12</a>
  </div>
  <p>Fork <code>compress/lzw</code>, fix #21-#23.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-05-01">2018-05-01</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.11">v0.1.11</a>
  </div>
  <p>Add LZWDecode filter support.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-04-17">2018-04-17</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.10">v0.1.10</a>
  </div>
  <p>Add name tree caching.<br> Fix #18.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-04-01">2018-04-01</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.9">v0.1.9</a>
  </div>
  <p>Redesign extraction API. <br>Fix #7.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-03-26">2018-03-26</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.8">v0.1.8</a>
  </div>
  <p>Introduce PDFObject interface.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-03-19">2018-03-19</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.7">v0.1.7</a>
  </div>
  <p>Add logging interface.<br> Merge PR #15.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-01-14">2018-01-14</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.6">v0.1.6</a>
  </div>
  <p>Add List/Add permissions command.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2018-01-08">2018-01-08</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.5">v0.1.5</a>
  </div>
  <p>Add Encrypt/Decrypt command.</p>
</article>


<h2 class="changelog-year">2017</h2>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2017-12-21">2017-12-21</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.4">v0.1.4</a>
  </div>
  <p>Fix object freelist management.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2017-12-12">2017-12-12</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.3">v0.1.3</a>
  </div>
  <p>Add Attachments command. Fix #9.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2017-11-27">2017-11-27</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.2">v0.1.2</a>
  </div>
  <p>Fix #11.</p>
</article>

<article class="changelog-entry">
  <div class="changelog-meta">
    <time datetime="2017-11-05">2017-11-05</time>
    <span class="changelog-kind">Release</span>
    <a href="https://github.com/pdfcpu/pdfcpu/releases/tag/v0.1.1">v0.1.1</a>
  </div>
  <p>Add examples.<br> Fix #10.</p>
</article>

</div>
