/*
Copyright 2020 The pdf Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

const (
	sampleArabic = `الإعلان العالمي لحقوق الإنسان

المادة 1.
يولد جميع الناس أحرارًا ومتساوين في الكرامة والحقوق.
وقد وهبوا عقلاً وضميرًا وعليهم أن يعامل بعضهم بعضًا بروح الإخاء.

المادة 2.
لكل فرد الحق في جميع الحقوق والحريات المنصوص عليها في هذا الإعلان ، دون تمييز من أي نوع ،
مثل العرق أو اللون أو الجنس أو اللغة أو الدين أو الرأي السياسي أو غير السياسي أو الأصل القومي أو الاجتماعي أو الملكية أو
ولادة أو حالة أخرى. علاوة على ذلك ، لا يجوز التمييز على أساس سياسي أو قضائي أو
الوضع الدولي للبلد أو الإقليم الذي ينتمي إليه الشخص ، سواء كان مستقلاً ، أو محل ثقة ،
غير متمتع بالحكم الذاتي أو تحت أي قيود أخرى على السيادة.`

	sampleArmenian = `Մարդու իրավունքների համընդհանուր հռչակագիր

Հոդված 1.
Բոլոր մարդիկ ծնվում են ազատ և հավասար ՝ արժանապատվությամբ և իրավունքներով:
Նրանք օժտված են բանականությամբ և խղճով և պետք է եղբայրության ոգով գործեն միմյանց նկատմամբ:

Հոդված 2.
Յուրաքանչյուր ոք ունի սույն Հռչակագրում ամրագրված բոլոր իրավունքներն ու ազատությունները առանց որևէ տեսակի տարբերակման,
ինչպիսիք են ռասան, գույնը, սեռը, լեզուն, կրոնը, քաղաքական կամ այլ կարծիքը, ազգային կամ սոցիալական ծագումը, սեփականությունը,
ծնունդ կամ այլ կարգավիճակ: Ավելին, որևէ տարբերակում չի կարող դրվել քաղաքական, իրավասության կամ իրավունքի հիման վրա
երկրի կամ տարածքի միջազգային կարգավիճակը, որին պատկանում է անձը, անկախ այն լինելուց, վստահություն,
ոչ ինքնակառավարվող կամ ինքնիշխանության որևէ այլ սահմանափակման ներքո:`

	sampleAzerbaijani = `Ümumdünya İnsan Haqları Bəyannaməsi

Maddə 1
Bütün insanlar azad və ləyaqət və hüquqlara bərabər olaraq doğulurlar.
Onlara ağıl və vicdan bəxş edilmişdir və bir-birlərinə qardaşlıq ruhunda davranmalıdırlar.

Maddə 2
Hər kəs bu Bəyannamədə göstərilən bütün hüquq və azadlıqlara, heç bir fərq qoymadan,
irqi, rəngi, cinsi, dili, dini, siyasi və ya digər fikri, milli və ya sosial mənşəyi, mülkiyyəti,
doğum və ya digər vəziyyət. Bundan əlavə, siyasi, yurisdiksiyalı və ya əsas götürülərək heç bir fərq qoyulmayacaqdır
bir şəxsin mənsub olduğu ölkənin və ya ərazinin beynəlxalq statusu, müstəqil, etibarlı,
özünüidarə etməmək və ya suverenliyin hər hansı digər məhdudiyyəti altında.`

	sampleBangla = `মানবাধিকারের সর্বজনীন ঘোষণা:

অনুচ্ছেদ 1।
সমস্ত মানুষ মর্যাদা ও অধিকারে স্বাধীন ও সমান জন্মগ্রহণ করে।
এগুলি যুক্তি ও বিবেকের অধিকারী এবং ভ্রাতৃত্বের মনোভাবের সাথে একে অপরের প্রতি আচরণ করা উচিত।

অনুচ্ছেদ 2।
এই ঘোষণাপত্রে নির্ধারিত সমস্ত অধিকার এবং স্বাধীনতার জন্য প্রত্যেকেই অধিকারপ্রাপ্ত,
কোনও প্রকারভেদ ছাড়াই, যেমন জাতি, বর্ণ, লিঙ্গ, ভাষা, ধর্ম, রাজনৈতিক বা অন্যান্য মতামত,
জাতীয় বা সামাজিক উত্স, সম্পত্তি, জন্ম বা অন্য অবস্থা। তদুপরি, রাজনৈতিক,
এখতিয়ার বা ভিত্তিতে কোনও পার্থক্য করা হবে না একজন ব্যক্তি যার দেশ বা অঞ্চলে তার আন্তর্জাতিক
অবস্থান, এটি স্বাধীন, বিশ্বাস, স্ব-শাসন পরিচালনা বা সার্বভৌমত্বের অন্য কোনও সীমাবদ্ধতার অধীনে।`

	sampleBelarusian = `УСЕАГУЛЬНАЯ ДЭКЛАРАЦЫЯ ПРАВОЎ ЧАЛАВЕКА

Артыкул 1.
Усе людзi нараджаюцца свабоднымi i роўнымi ў сваёй годнасцi i правах. Яны надзелены розумам i сумленнем i павiнны ставiцца
адзiн да аднаго ў духу брацтва.

Артыкул 2.
Кожны чалавек павiнен валодаць усiмi правамi i ўсiмi свабодамi, што абвешчаны гэтай Дэкларацыяй, без якога б там нi было адрознення,
як напрыклад у адносiнах расы, колеру скуры, полу, мовы, рэлiгii, палiтычных або iншых перакананняў, нацыянальнага або сацыяльнага
паходжання, маёмаснага, саслоўнага або iншага становiшча.
Апрача таго, не павiнна рабiцца нiякага адрознення на аснове палiтычнага, прававога або мiжнароднага статуса краiны або тэрыторыi,
да якой чалавек належыць, незалежна ад таго, цi з’яўляецца гэта тэрыторыя незалежнай, падапечнай, несамакiравальнай,
або як-небудзь iнакш абмежаванай у сваiм суверэнiтэце.`

	sampleChineseSimple = `世界人权宣言

第一条
人人生而自由,在尊严和权利上一律平等。他们赋有理性和良心,并应以兄弟关系的精神相对待。

第二条
人人有资格享有本宣言所载的一切权利和自由,不分种族、肤色、性别、语言、宗教、政治或其他见解、国籍或社会出身、财产、出生或其他身分等任何区别。

并且不得因一人所属的国家或领土的政治的、行政的或者国际的地位之不同而有所区别,无论该领土是独立领土、托管领土、非自治领土或者处于其他任何主权受限制的情况之下。`

	sampleChineseTraditional = `世界人權宣言

第一條
人人生而自由,在尊嚴和權利上一律平等。他們賦有理性和良心,並應以兄弟關係的精神相對待。

第二條
人人有資格享有本宣言所載的一切權利和自由,不分種族、膚色、性別、語言、宗教、政治或其他見解、國籍或社會出身、財產、出生或其他身分等任何區別。

並且不得因一人所屬的國家或領土的政治的、行政的或者國際的地位之不同而有所區別,無論該領土是獨立領土、託管領土、非自治領土或者處於其他任何主權受限制的情況之下。`

	sampleEnglish = `Universal Declaration of Human Rights

Article 1.
All human beings are born free and equal in dignity and rights.
They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood.

Article 2.
Everyone is entitled to all the rights and freedoms set forth in this Declaration, without distinction of any kind,
such as race, colour, sex, language, religion, political or other opinion, national or social origin, property,
birth or other status. Furthermore, no distinction shall be made on the basis of the political, jurisdictional or
international status of the country or territory to which a person belongs, whether it be independent, trust,
non-self-governing or under any other limitation of sovereignty.`

	sampleFrench = `Déclaration universelle des droits de l'Homme

Article 1.
Tous les êtres humains naissent libres et égaux en dignité et en droits. Ils sont doués de raison et de conscience et doivent agir les uns envers les autres
dans un esprit de fraternité.

Article 2.
Chacun peut se prévaloir de tous les droits et de toutes les libertés proclamés dans la présente Déclaration, sans distinction aucune, notamment de race,
de couleur, de sexe, de langue, de religion, d'opinion politique ou de toute autre opinion, d'origine nationale ou sociale, de fortune, de naissance ou de toute
autre situation. De plus, il ne sera fait aucune distinction fondée sur le statut politique, juridique ou international du pays ou du territoire dont une personne
est ressortissante, que ce pays ou territoire soit indépendant, sous tutelle, non autonome ou soumis à une limitation quelconque de souveraineté.`

	sampleGerman = `Die Allgemeine Erklärung der Menschenrechte

	Artikel 1.
Alle Menschen sind frei und gleich an Würde und Rechten geboren.
Sie sind mit Vernunft und Gewissen begabt und sollen einander im Geist der Brüderlichkeit begegnen.

Artikel 2.
Jeder hat Anspruch auf die in dieser Erklärung verkündeten Rechte und Freiheiten ohne irgendeinen Unterschied,
etwa nach Rasse, Hautfarbe, Geschlecht, Sprache, Religion, politischer oder sonstiger Überzeugung, nationaler
oder sozialer Herkunft, Vermögen, Geburt oder sonstigem Stand.
Des weiteren darf kein Unterschied gemacht werden auf Grund der politischen, rechtlichen oder internationalen
Stellung des Landes oder Gebiets, dem eine Person angehört, gleichgültig ob dieses unabhängig ist, unter Treuhandschaft steht,
keine Selbstregierung besitzt oder sonst in seiner Souveränität eingeschränkt ist.`

	sampleGreek = `ΟΙΚΟΥΜΕΝΙΚΗ ΔΙΑΚΗΡΥΞΗ ΓΙΑ ΤΑ ΑΝΘΡΩΠΙΝΑ ΔΙΚΑΙΩΜΑΤΑ

ΑΡΘΡΟ 1
Ολοι οι άνθρωποι γεννιούνται ελεύθεροι και ίσοι στην αξιοπρέπεια και τα δικαιώματα.
Είναι προικισμένοι με λογική και συνείδηση, και οφείλουν να συμπεριφέρονται μεταξύ τους με πνεύμα αδελφοσύνης.

ΑΡΘΡΟ 2
Κάθε άνθρωπος δικαιούται να επικαλείται όλα τα δικαιώματα και όλες τις ελευθερίες που προκηρύσσει η παρούσα Διακήρυξη,
χωρίς καμία απολύτως διάκριση, ειδικότερα ως προς τη φυλή, το χρώμα, το φύλο, τη γλώσσα, τις θρησκείες, τις πολιτικές
ή οποιεσδήποτε άλλες πεποιθήσεις, την εθνική ή κοινωνική καταγωγή, την περιουσία, τη γέννηση ή οποιαδήποτε άλλη κατάσταση.`

	sampleHebrew = `הצהרה האוניברסלית של זכויות האדם

מאמר 1.
כל בני האדם נולדים חופשיים ושווים בכבודם ובזכויותיהם.
הם ניחנים בתבונה ובמצפון ועליהם לפעול זה כלפי זה ברוח אחווה.

סעיף 2.
כל אחד זכאי לכל הזכויות והחירויות המופיעות בהצהרה זו, ללא הבחנה מכל סוג שהוא,
כגון גזע, צבע, מין, שפה, דת, דעה פוליטית או אחרת, מקור לאומי או חברתי, רכוש,
לידה או מעמד אחר. יתר על כן, לא תיעשה הבחנה על בסיס הפוליטי, השיפוט או
מעמד בינלאומי של המדינה או השטח שאדם משתייך אליו, בין אם זה עצמאי, אמון,
לא ממשל עצמי או תחת כל מגבלה אחרת של ריבונות.`

	sampleHindi = `मानव अधिकारों की सार्वभौम घोषणा

लेख 1।
सभी मनुष्यों का जन्म स्वतंत्र और समान सम्मान और अधिकार में हुआ है।
वे तर्क और विवेक के साथ संपन्न होते हैं और भाईचारे की भावना से एक दूसरे के प्रति कार्य करना चाहिए।

अनुच्छेद 2।
हर कोई इस घोषणा में उल्लिखित सभी अधिकारों और स्वतंत्रता का हकदार है, किसी भी प्रकार का भेद किए बिना,
जैसे कि जाति, रंग, लिंग, भाषा, धर्म, राजनीतिक या अन्य मत, राष्ट्रीय या सामाजिक मूल, संपत्ति,
जन्म या अन्य स्थिति। इसके अलावा, राजनीतिक, अधिकार क्षेत्र या के आधार पर कोई भेद नहीं किया जाएगा
देश या क्षेत्र की अंतर्राष्ट्रीय स्थिति, जो किसी व्यक्ति की है, चाहे वह स्वतंत्र हो, भरोसा हो,
स्व-शासन या संप्रभुता के किसी अन्य सीमा के तहत।`

	sampleHungarian = `Az Emberi Jogok Egyetemes Nyilatkozata

1. cikk
Minden. emberi lény szabadon születik és egyenlő méltósága és joga van. Az emberek, ésszel és lelkiismerettel bírván,
egymással szemben testvéri szellemben kell hogy viseltessenek.

2. cikk
Mindenki, bármely megkülönböztetésre, nevezetesen fajra, színre, nemre, nyelvre, vallásra, politikai vagy bármely
más véleményre, nemzeti vagy társadalmi eredetre, vagyonra, születésre, vagy bármely más körülményre való tekintet
nélkül hivatkozhat a jelen Nyilatkozatban kinyilvánított összes jogokra és szabadságokra.
Ezenfelül nem lehet semmiféle megkülönböztetést tenni annak az országnak, vagy területnek politikai, jogi vagy
nemzetközi helyzete alapján sem, amelynek a személy állampolgára, aszerint, hogy az illető ország vagy terület független,
gyámság alatt áll, nem autonóm vagy szuverenitása bármely vonatkozásban korlátozott.`

	sampleIndonesian = `Pernyataan Umum tentang Hak-Hak Asasi Manusia

Pasal 1
Semua orang dilahirkan merdeka dan mempunyai martabat dan hak-hak yang sama.
Mereka dikaruniai akal dan hati nurani dan hendaknya bergaul satu sama lain dalam semangat persaudaraan.

Pasal 2
Setiap orang berhak atas semua hak dan kebebasan-kebebasan yang tercantum di dalam Pernyataan ini tanpa perkecualian apapun,
seperti ras, warna kulit, jenis kelamin, bahasa, agama, politik atau pendapat yang berlainan, asal mula kebangsaan atau
kemasyarakatan, hak milik, kelahiran ataupun kedudukan lain.
Di samping itu, tidak diperbolehkan melakukan perbedaan atas dasar kedudukan politik, hukum atau kedudukan internasional
dari negara atau daerah dari mana seseorang berasal, baik dari negara yang merdeka, yang berbentuk wilayah-wilayah perwalian,
jajahan atau yang berada di bawah batasan kedaulatan yang lain.`

	sampleItalian = `DICHIARAZIONE UNIVERSALE DEI DIRITTI UMANI

Articolo 1
Tutti gli esseri umani nascono liberi ed eguali in dignità e diritti.
Essi sono dotati di ragione e di coscienza e devono agire gli uni verso gli altri in spirito di fratellanza.

Articolo 2
Ad ogni individuo spettano tutti i diritti e tutte le libertà enunciate nella presente Dichiarazione,
senza distinzione alcuna, per ragioni di razza, di colore, di sesso, di lingua, di religione,
di opinione politica o di altro genere, di origine nazionale o sociale, di ricchezza, di nascita o di altra condizione.
Nessuna distinzione sarà inoltre stabilita sulla base dello statuto politico, giuridico o internazionale del paese
o del territorio cui una persona appartiene, sia indipendente, o sottoposto ad amministrazione fiduciaria o non autonomo,
o soggetto a qualsiasi limitazione di sovranità.`

	sampleJapanese = `『世界人権宣言』

第１条

すべての人間は、生まれながらにして自由であり、かつ、尊厳と権利と について平等である。人間は、理性と良心とを授けられており、互いに同 胞の精神をもって行動しなければならない。

第２条

すべて人は、人種、皮膚の色、性、言語、宗教、政治上その他の意見、　　　
国民的もしくは社会的出身、財産、門地その他の地位又はこれに類するい　　　
かなる自由による差別をも受けることなく、この宣言に掲げるすべての権　　　
利と自由とを享有することができる。

さらに、個人の属する国又は地域が独立国であると、信託統治地域で　　　
あると、非自治地域であると、又は他のなんらかの主権制限の下にあると　　　
を問わず、その国又は地域の政治上、管轄上又は国際上の地位に基ずくい　　　
かなる差別もしてはならない。`

	sampleKorean = `세 계 인 권 선 언

제 1 조
모든 인간은 태어날 때부터 자유로우며 그 존엄과 권리에 있어 동등하다. 인간은 천부적으로 이성과 양심을 부여받았으며 서로 형제애의 정신으로 행동하여야 한다.

제 2 조
모든 사람은 인종, 피부색, 성, 언어, 종교, 정치적 또는 기타의 견해, 민족적 또는 사회적 출신, 재산, 출생 또는 기타의 신분과 같은 어떠한 종류의 차별이 없이,
이 선언에 규정된 모든 권리와 자유를 향유할 자격이 있다 . 더 나아가 개인이 속한 국가 또는 영토가 독립국 , 신탁통치지역 , 비자치지역이거나 또는 주권에 대한 여타의 제약을 받느냐에 관계없이 ,
그 국가 또는 영토의 정치적, 법적 또는 국제적 지위에 근거하여 차별이 있어서는 아니된다 .`

	sampleMarathi = `मानवाधिकारांची सार्वत्रिक घोषणा

अनुच्छेद १.
सर्व मानव स्वतंत्र आणि समान सन्मान आणि अधिकारात जन्माला येतात. ते तर्क आणि विवेकबुद्धीने संपन्न आहेत आणि त्यांनी बंधुत्वाच्या भावनेने एकमेकांशी वागले पाहिजे.

अनुच्छेद २.
या घोषणेमध्ये वंश, रंग, लिंग, भाषा, धर्म, राजकीय किंवा अन्य मत, राष्ट्रीय किंवा सामाजिक मूळ, मालमत्ता, जन्म किंवा इतर कोणत्याही प्रकारचे भेद न
करता प्रत्येकजण या घोषणेत नमूद केलेले सर्व अधिकार आणि स्वातंत्र्य मिळविण्यास पात्र आहे. इतर स्थिती. याव्यतिरिक्त, एखादा देश ज्याच्या ताब्यात आहे तो
देशाच्या राजकीय, कार्यकक्षात्मक किंवा आंतरराष्ट्रीय दर्जाच्या आधारे कोणताही भेदभाव केला जाणार नाही, तो स्वतंत्र, विश्वास असो, स्वराज्य असो किंवा
सार्वभौमत्वाच्या कोणत्याही अन्य मर्यादेखाली असो.`

	samplePersian = `اعلامیه جهانی حقوق بشر

مقاله 1.
همه انسانها آزاد و از نظر کرامت و حقوق برابر به دنیا می آیند.
آنها از عقل و وجدان برخوردارند و باید با روحیه برادری نسبت به یکدیگر رفتار کنند.

ماده 2
هر کس بدون هیچ گونه تمایزی از کلیه حقوق و آزادی های مندرج در این بیانیه برخوردار است ،
مانند نژاد ، رنگ ، جنس ، زبان ، مذهب ، عقاید سیاسی یا عقاید دیگر ، منشا national ملی یا اجتماعی ، دارایی ،
تولد یا وضعیت دیگر بعلاوه ، هیچ تفکیکی نباید بر اساس حوزه های سیاسی ، قضایی یا قضایی قائل شود
وضعیت بین المللی کشور یا سرزمینی که شخص به آن تعلق دارد ، خواه استقلال باشد ،
غیر خود حاکم یا تحت هر محدودیت دیگری در حاکمیت.`

	samplePolish = `POWSZECHNA DEKLARACJA PRAW CZŁOWIEKA

Artykuł 1
Wszyscy ludzie rodzą się wolni i równi pod względem swej godności i swych praw. Są oni obdarzeni rozumem i sumieniem
i powinni postępować wobec innych w duchu braterstwa.

Artykuł 2
Każdy człowiek posiada wszystkie prawa i wolności zawarte w niniejszej Deklaracji bez względu na jakiekolwiek różnice rasy,
koloru, płci, języka, wyznania, poglądów politycznych i innych, narodowości, pochodzenia społecznego, majątku,
urodzenia lub jakiegokolwiek innego stanu.
Nie wolno ponadto czynić żadnej różnicy w zależności od sytuacji politycznej, prawnej lub międzynarodowej kraju lub obszaru,
do którego dana osoba przynależy, bez względu na to, czy dany kraj lub obszar jest niepodległy, czy też podlega systemowi
powiernictwa, nie rządzi się samodzielnie lub jest w jakikolwiek sposób ograniczony w swej niepodległości.`

	samplePortuguese = `Declaração Universal dos Direitos Humanos

Artigo 1°
Todos os seres humanos nascem livres e iguais em dignidade e em direitos. Dotados de razão e de consciência,
devem agir uns para com os outros em espírito de fraternidade.	

Artigo 2°
Todos os seres humanos podem invocar os direitos e as liberdades proclamados na presente Declaração,
sem distinção alguma, nomeadamente de raça, de cor, de sexo, de língua, de religião, de opinião política ou outra,
de origem nacional ou social, de fortuna, de nascimento ou de qualquer outra situação.
Além disso, não será feita nenhuma distinção fundada no estatuto político, jurídico ou internacional do país ou do
território da naturalidade da pessoa, seja esse país ou território independente, sob tutela, autônomo ou sujeito
a alguma limitação de soberania.`

	sampleRussian = `Всеобщая декларация прав человека

Статья 1
Все люди рождаются свободными и равными в своем достоинстве и правах.
Они наделены разумом и совестью и должны поступать в отношении друг друга в духе братства.

Статья 2
Каждый человек должен обладать всеми правами и всеми свободами, провозглашенными настоящей
Декларацией, без какого бы то ни было различия, как-то в отношении расы, цвета кожи, пола,
языка, религии, политических или иных убеждений, национального или социального происхождения,
имущественного, сословного или иного положения.
Кроме того, не должно проводиться никакого различия на основе политического, правового или
международного статуса страны или территории, к которой человек принадлежит, независимо от того,
является ли эта территория независимой, подопечной, несамоуправляющейся или как-либо иначе
ограниченной в своем суверенитете.`

	sampleSpanish = `Declaración Universal de Derechos Humanos

Artículo 1.
Todos los seres humanos nacen libres e iguales en dignidad y derechos y, dotados como están de razón y conciencia,
deben comportarse fraternalmente los unos con los otros.

Artículo 2.
Toda persona tiene los derechos y libertades proclamados en esta Declaración, sin distinción alguna de raza, color,
sexo, idioma, religión, opinión política o de cualquier otra índole, origen nacional o social, posición económica,
nacimiento o cualquier otra condición. Además, no se hará distinción alguna fundada en la condición política,jurídica
o internacional del país o territorio de cuya jurisdicción dependa una persona, tanto si se trata de un país independiente,
como de un territorio bajo administración fiduciaria, no autónomo o sometido a cualquier otra limitación de soberanía.`

	sampleSwahili = `UMOJA WA MATAIFA OFISI YA IDARA YA HABARI TAARIFA YA ULIMWENGU JUU YA HAKI ZA BINADAMU

Kifungu cha 1.
Watu wote wamezaliwa huru, hadhi na haki zao ni sawa. Wote wamejaliwa akili na dhamiri, hivyo yapasa watendeane kindugu.

Kifungu cha 2.
Kila mtu anastahili kuwa na haki zote na uhuru wote ambao umeelezwa katika Taarifa hii bila ubaguzi wo wote. Yaani bila kubaguana kwa rangi,
taifa, wanaume kwa wanawake, dini, siasa, fikara, asili ya taifa la mtu, mali, kwa kizazi au kwa hali nyingine yo yote.
Juu ya hayo usifanye ubaguzi kwa kutegemea siasa, utawala au kwa kutegemea uhusiano wa nchi fulani na mataifa mengine au nchi ya asili ya mtu,
haidhuru nchi hiyo iwe inayojitawala, ya udhamini, isiyojitawala au inayotawaliwa na nchi nyingine kwa hali ya namna yo yote.`

	sampleSwedish = `ALLMÄN FÖRKLARING OM DE MÄNSKLIGA RÄTTIGHETERNA

Artikel 1.
Alla människor äro födda fria och lika i värde och rättigheter.
De äro utrustade med förnuft och samvete och böra handla gentemot varandra i en anda av broderskap.

Artikel 2.
Envar är berättigad till alla de fri- och rättigheter, som uttalas i denna förklaring, utan åtskillnad av något slag,
såsom ras, hudfärg, kön, språk, religion, politisk eller annan uppfattning, nationellt eller socialt ursprung,
egendom, börd eller ställning i övrigt.
Ingen åtskillnad må vidare göras på grund av den politiska, juridiska eller internationella ställning, som intages 
v det land eller område, till vilket en person hör, vare sig detta land eller område är oberoende, står under
förvaltarskap, är icke-självstyrande eller är underkastat någon annan begränsning av sin suveränitet.`

	sampleThai = `ปฏิญญาสากลว่าด้วยสิทธิมนุษยชน

หัวข้อที่ 1.
มนุษย์ทุกคนเกิดมาโดยเสรีและเท่าเทียมกันในศักดิ์ศรีและสิทธิ
พวกเขากอปรด้วยเหตุผลและมโนธรรมและควรปฏิบัติต่อกันด้วยจิตวิญญาณแห่งความเป็นพี่น้องกัน

ข้อ 2.
ทุกคนมีสิทธิได้รับสิทธิและเสรีภาพทั้งหมดที่กำหนดไว้ในปฏิญญานี้โดยไม่มีความแตกต่างใด ๆ
เช่นเชื้อชาติสีผิวเพศภาษาศาสนาความคิดเห็นทางการเมืองหรืออื่น ๆ ชาติกำเนิดหรือสังคมทรัพย์สิน
การเกิดหรือสถานะอื่น ๆ นอกจากนี้จะไม่มีการสร้างความแตกต่างใด ๆ บนพื้นฐานของการเมืองเขตอำนาจศาลหรือ
สถานะระหว่างประเทศของประเทศหรือดินแดนที่บุคคลเป็นอยู่ไม่ว่าจะเป็นอิสระความไว้วางใจ
ไม่ปกครองตนเองหรืออยู่ภายใต้ข้อ จำกัด อื่นใดของอำนาจอธิปไตย`

	sampleTurkish = `İnsan hakları  evrensel beyannamesi

Madde 1
Bütün insanlar hür, haysiyet ve haklar bakımından eşit doğarlar.
Akıl ve vicdana sahiptirler ve birbirlerine karşı kardeşlik zihniyeti ile hareket etmelidirler.

Madde 2
Herkes, ırk, renk, cinsiyet, dil, din, siyasi veya diğer herhangi bir akide, milli veya içtimai menşe,
servet, doğuş veya herhangi diğer bir fark gözetilmeksizin işbu Beyannamede ilan olunan tekmil haklardan
ve bütün hürriyetlerden istifade edebilir.
Bundan başka, bağımsız memleket uyruğu olsun, vesayet altında bulunan, gayri muhtar veya sair bir egemenlik
kayıtlamasına tabi ülke uyruğu olsun, bir şahıs hakkında, uyruğu bulunduğu memleket veya ülkenin siyasi,
hukuki veya milletlerarası statüsü bakımından hiçbir ayrılık gözetilmeyecektir.`

	sampleUrdu = `انسانی حقوق کا عالمی اعلان

آرٹیکل 1۔
تمام انسان وقار اور حقوق میں آزاد اور برابر پیدا ہوئے ہیں۔
وہ استدلال اور ضمیر کے مالک ہیں اور بھائی چارے کے جذبے سے ایک دوسرے کے ساتھ کام کریں۔

آرٹیکل 2۔
ہر شخص کسی بھی طرح کے امتیاز کے بغیر ، اس اعلامیے میں بیان کردہ تمام حقوق اور آزادی کا حقدار ہے ،
جیسے نسل ، رنگ ، جنس ، زبان ، مذہب ، سیاسی یا دوسری رائے ، قومی یا معاشرتی اصل ، املاک ،
پیدائش یا دوسری حیثیت مزید برآں ، سیاسی ، دائرہ اختیار یا کی بنیاد پر کوئی امتیاز نہیں برپا کیا جائے گا
ملک یا علاقے کی بین الاقوامی حیثیت جس سے کسی شخص کا تعلق ہے ، خواہ وہ آزاد ہو ، اعتماد ،
غیر خود حکمرانی یا خود مختاری کی کسی بھی دوسری حد کے تحت۔`

	sampleVietnamese = `Tuyên ngôn nhân quyền

Điều 1.
Tất cả con người sinh ra đều tự do, bình đẳng về nhân phẩm và quyền.
Họ được phú cho lý trí và lương tâm và nên hành động với nhau trong tinh thần anh em.

Điều 2.
Mọi người được hưởng tất cả các quyền và tự do được nêu trong Tuyên bố này, không phân biệt bất kỳ
hình thức nào, chẳng hạn như chủng tộc, màu da, giới tính, ngôn ngữ, tôn giáo, chính trị hoặc quan
điểm khác, nguồn gốc quốc gia hoặc xã hội, tài sản, nơi sinh hoặc trạng thái khác.
Hơn nữa, không có sự phân biệt nào được thực hiện trên cơ sở địa vị chính trị, quyền tài phán hoặc
quốc tế của quốc gia hoặc vùng lãnh thổ mà một người thuộc về, cho dù đó là quốc gia độc lập,
tin cậy, không tự quản hay theo bất kỳ giới hạn chủ quyền nào khác.`
)

type sample struct {
	fontName string
	lang     string
	text     string
	rtl      bool
}

var langSamples = []sample{
	{"UnifontMedium", "Arabic", sampleArabic, true},
	{"UnifontMedium", "Armenian", sampleArmenian, false},
	{"Roboto-Regular", "Azerbaijani", sampleAzerbaijani, false},
	{"UnifontMedium", "Bangla", sampleBangla, false},
	{"Roboto-Regular", "Belarusian", sampleBelarusian, false},
	{"UnifontMedium", "Chinese simple", sampleChineseSimple, false},
	{"UnifontMedium", "Chinese traditional", sampleChineseTraditional, false},
	{"Roboto-Regular", "English", sampleEnglish, false},
	{"Roboto-Regular", "French", sampleFrench, false},
	{"Roboto-Regular", "German", sampleGerman, false},
	{"Roboto-Regular", "Greek", sampleGreek, false},
	{"UnifontMedium", "Hebrew", sampleHebrew, true},
	{"UnifontMedium", "Hindi", sampleHindi, false},
	{"Roboto-Regular", "Hungarian", sampleHungarian, false},
	{"Roboto-Regular", "Indonesian", sampleIndonesian, false},
	{"Roboto-Regular", "Italian", sampleItalian, false},
	{"Unifont-JPMedium", "Japanese", sampleJapanese, false},
	{"UnifontMedium", "Korean", sampleKorean, false},
	{"UnifontMedium", "Marathi", sampleMarathi, false},
	{"UnifontMedium", "Persian", samplePersian, true},
	{"Roboto-Regular", "Portuguese", samplePortuguese, false},
	{"Roboto-Regular", "Polish", samplePolish, false},
	{"Roboto-Regular", "Russian", sampleRussian, false},
	{"Roboto-Regular", "Spanish", sampleSpanish, false},
	{"Roboto-Regular", "Swahili", sampleSwahili, false},
	{"Roboto-Regular", "Swedish", sampleSwedish, false},
	{"UnifontMedium", "Thai", sampleThai, false},
	{"Roboto-Regular", "Turkish", sampleTurkish, false},
	{"UnifontMedium", "Urdu", sampleUrdu, true},
	{"Roboto-Regular", "Vietnamese", sampleVietnamese, false},
}

func renderArticle(p pdf.Page, row, col, lang int) {
	mediaBox := p.MediaBox
	w := mediaBox.Width() / 6
	h := mediaBox.Height() / 5
	region := pdf.RectForWidthAndHeight(float64(col)*w, float64(4-row)*h, w, h)
	buf := p.Buf
	sample := langSamples[lang]

	if lang%2 > 0 {
		pdf.FillRect(buf, region, pdf.SimpleColor{R: .75, G: .75, B: 1})
	}

	fontName := "Helvetica"
	k := p.Fm.EnsureKey("Helvetica")

	td := pdf.TextDescriptor{
		Text:           sample.lang,
		FontName:       fontName,
		FontKey:        k,
		FontSize:       24,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		Scale:          1.,
		ScaleAbs:       false,
		HAlign:         pdf.AlignLeft,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     true,
		HairCross:      false,
	}

	pdf.WriteColumnAnchored(buf, mediaBox, region, td, pdf.TopLeft, 0)

	fontName = sample.fontName
	k = p.Fm.EnsureKey(fontName)

	td = pdf.TextDescriptor{
		Text:           sample.text,
		FontName:       fontName,
		RTL:            sample.rtl,
		FontKey:        k,
		FontSize:       16,
		MLeft:          5,
		MRight:         5,
		MTop:           5,
		MBot:           5,
		X:              -1,
		Y:              -1,
		Scale:          .9,
		ScaleAbs:       false,
		HAlign:         pdf.AlignJustify,
		VAlign:         pdf.AlignMiddle,
		RMode:          pdf.RMFill,
		StrokeCol:      pdf.NewSimpleColor(0x206A29),
		FillCol:        pdf.NewSimpleColor(0x206A29),
		ShowBackground: true,
		BackgroundCol:  pdf.SimpleColor{R: 1., G: .98, B: .77},
		ShowBorder:     true,
		ShowLineBB:     false,
		ShowTextBB:     false,
		HairCross:      false,
	}

	if sample.lang == "Japanese" {
		pdf.WriteColumn(buf, mediaBox, region, td, mediaBox.Width()*.9)
		return
	}

	if sample.lang == "Thai" {
		td.HAlign = pdf.AlignLeft
		pdf.WriteColumn(buf, mediaBox, region, td, mediaBox.Width()*.9)
		return
	}

	pdf.WriteMultiLine(buf, mediaBox, region, td)
}

func TestUserFonts(t *testing.T) {
	msg := "TestUserFonts"

	api.LoadConfiguration()
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	w, h := 600., 600.
	mediaBox := pdf.RectForDim(w, h)
	p := pdf.NewPageWithBg(mediaBox, pdf.NewSimpleColor(0xbeded9))

	lang := 0
	for row := 0; row < 5; row++ {
		for col := 0; col < 6; col++ {
			renderArticle(p, row, col, lang)
			lang++
		}
	}

	createXRefAndWritePDF(t, msg, "UserFont_HumanRights", p)
}
