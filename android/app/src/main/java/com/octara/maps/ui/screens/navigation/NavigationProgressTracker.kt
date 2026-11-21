package com.octara.maps.ui.screens.navigation

import com.octara.maps.types.LegType
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.TripLeg
import com.octara.maps.ui.components.distanceToString
import com.octara.maps.ui.components.formatTimeString
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.asin
import kotlin.math.cos
import kotlin.math.hypot
import kotlin.math.max
import kotlin.math.min
import kotlin.math.sin
import kotlin.math.sqrt

data class LegPlatformHint(
    val ref: String? = null,
    val exitSide: String? = null
)

data class NavigationProgress(
    val activeLegIndex: Int,
    val activeStopPosition: Int?,
    val isAtStop: Boolean,
    val isAlightingSoon: Boolean,
    val stopsRemaining: Int?,
    val nextActionTitle: String,
    val nextActionSubtitle: String?,
    val remainingMinutes: Int,
    val arrivalTimeText: String,
    val adjustedLegs: List<TripLeg>,
    val gpsSynced: Boolean
)

data class StopNode(
    val lat: Double,
    val lon: Double,
    val arrMillis: Long?,
    val depMillis: Long?
)

class LegGpsTrackState {
    var lastAtStop: Int? = null
    var lastAtStopMillis: Long = 0L
    var furthestStop: Int = 0
    var hasLeftStart: Boolean = false
    val visitedStops: MutableSet<Int> = mutableSetOf()
    val observedDepartures: MutableMap<Int, Long> = mutableMapOf()
    val plannedAtDeparture: MutableMap<Int, Long> = mutableMapOf()

    fun reset() {
        lastAtStop = null
        lastAtStopMillis = 0L
        furthestStop = 0
        hasLeftStart = false
        visitedStops.clear()
        observedDepartures.clear()
        plannedAtDeparture.clear()
    }
}

class NavigationSessionTracker {
    var maxReachedLegIndex: Int = 0
    val legStates: MutableMap<Int, LegGpsTrackState> = mutableMapOf()

    fun stateForLeg(legIndex: Int): LegGpsTrackState =
        legStates.getOrPut(legIndex) { LegGpsTrackState() }

    fun resetForResync() {
        maxReachedLegIndex = 0
        legStates.values.forEach { it.reset() }
    }
}

internal fun parseIsoTimeMillis(time: String?): Long? {
    if (time.isNullOrBlank()) return null
    val trimmed = time.trim()
    return runCatching { OffsetDateTime.parse(trimmed).toInstant().toEpochMilli() }.getOrNull()
        ?: runCatching { ZonedDateTime.parse(trimmed).toInstant().toEpochMilli() }.getOrNull()
        ?: runCatching { Instant.parse(trimmed).toEpochMilli() }.getOrNull()
        ?: runCatching {
            LocalDateTime.parse(trimmed).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
        }.getOrNull()
        ?: runCatching {
            LocalTime.parse(trimmed).atDate(LocalDate.now()).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()
        }.getOrNull()
}

private val ISO_FORMATTER: DateTimeFormatter = DateTimeFormatter.ISO_OFFSET_DATE_TIME

internal fun formatMillisLike(template: String?, millis: Long): String {
    if (!template.isNullOrBlank()) {
        val trimmed = template.trim()
        runCatching {
            val odt = OffsetDateTime.parse(trimmed)
            return Instant.ofEpochMilli(millis).atOffset(odt.offset).format(ISO_FORMATTER)
        }
        runCatching {
            val zdt = ZonedDateTime.parse(trimmed)
            return Instant.ofEpochMilli(millis).atZone(zdt.zone).format(ISO_FORMATTER)
        }
    }
    return Instant.ofEpochMilli(millis).atZone(ZoneId.systemDefault()).format(ISO_FORMATTER)
}

internal fun geoDistanceMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Float {
    val r = 6371000.0
    val dLat = (lat2 - lat1) * PI / 180.0
    val dLon = (lon2 - lon1) * PI / 180.0
    val rLat1 = lat1 * PI / 180.0
    val rLat2 = lat2 * PI / 180.0
    val a = sin(dLat / 2) * sin(dLat / 2) +
        cos(rLat1) * cos(rLat2) * sin(dLon / 2) * sin(dLon / 2)
    val c = 2.0 * asin(min(1.0, sqrt(a)))
    return (r * c).toFloat()
}

private data class SegProj(val t: Double, val distMeters: Float)

private fun projectPointToSegment(
    pLat: Double,
    pLon: Double,
    aLat: Double,
    aLon: Double,
    bLat: Double,
    bLon: Double
): SegProj {
    val refLat = (aLat + bLat) * 0.5
    val cosLat = cos(refLat * PI / 180.0)
    val mPerDeg = 111320.0
    val ax = (aLon - pLon) * cosLat * mPerDeg
    val ay = (aLat - pLat) * mPerDeg
    val bx = (bLon - pLon) * cosLat * mPerDeg
    val by = (bLat - pLat) * mPerDeg
    val vx = bx - ax
    val vy = by - ay
    val len2 = vx * vx + vy * vy
    val t = if (len2 > 1e-6) {
        (-(ax * vx + ay * vy) / len2).coerceIn(0.0, 1.0)
    } else {
        0.0
    }
    val dx = ax + t * vx
    val dy = ay + t * vy
    return SegProj(t = t, distMeters = hypot(dx, dy).toFloat())
}

internal fun buildLegStopNodes(leg: TripLeg): List<StopNode> {
    val list = mutableListOf<StopNode>()
    val dep0 = parseIsoTimeMillis(leg.departureTime)
    list.add(StopNode(leg.fromLat, leg.fromLon, arrMillis = dep0, depMillis = dep0))
    leg.intermediateStops.orEmpty().forEach { s ->
        val rawArr = s.arrivalTime.ifBlank { s.departureTime }
        val rawDep = s.departureTime.ifBlank { s.arrivalTime }
        list.add(StopNode(s.lat, s.lon, arrMillis = parseIsoTimeMillis(rawArr), depMillis = parseIsoTimeMillis(rawDep)))
    }
    val arrN = parseIsoTimeMillis(leg.arrivalTime)
    list.add(StopNode(leg.toLat, leg.toLon, arrMillis = arrN, depMillis = arrN))
    return list
}

internal fun legCorridorDistanceMeters(leg: TripLeg, lat: Double, lon: Double): Float? {
    var best: Float? = null
    fun consider(d: Float) {
        if (best == null || d < best!!) best = d
    }

    val nodes = buildLegStopNodes(leg)
    for (n in nodes) {
        if (n.lat != 0.0 || n.lon != 0.0) {
            consider(geoDistanceMeters(lat, lon, n.lat, n.lon))
        }
    }
    for (i in 0 until nodes.lastIndex) {
        val a = nodes[i]
        val b = nodes[i + 1]
        if ((a.lat != 0.0 || a.lon != 0.0) && (b.lat != 0.0 || b.lon != 0.0)) {
            consider(projectPointToSegment(lat, lon, a.lat, a.lon, b.lat, b.lon).distMeters)
        }
    }
    val coords = leg.coordinates
    if (coords != null && coords.size >= 2) {
        for (i in 0 until coords.lastIndex) {
            val a = coords[i]
            val b = coords[i + 1]
            if (a.size >= 2 && b.size >= 2) {
                consider(projectPointToSegment(lat, lon, a[0], a[1], b[0], b[1]).distMeters)
            }
        }
    }
    return best
}

private const val STOP_ENTER_RADIUS_M = 165f
private const val STOP_EXIT_RADIUS_M = 195f
private const val CORRIDOR_LOCK_RADIUS_M = 650f
private const val LIGHT_OFFSET_MAX_MS = 6 * 60_000L

private data class GpsStopMatch(
    val stopPosition: Int,
    val isAtStop: Boolean,
    val leftStartByGps: Boolean
)

private fun updateAndResolveTransitGps(
    leg: TripLeg,
    nodes: List<StopNode>,
    lat: Double,
    lon: Double,
    nowMillis: Long,
    state: LegGpsTrackState
): GpsStopMatch? {
    if (nodes.size < 2) return null

    var closestIdx = -1
    var closestDist = Float.MAX_VALUE
    nodes.forEachIndexed { idx, n ->
        if (n.lat != 0.0 || n.lon != 0.0) {
            val d = geoDistanceMeters(lat, lon, n.lat, n.lon)
            if (d < closestDist) {
                closestDist = d
                closestIdx = idx
            }
        }
    }
    if (closestIdx == -1) return null

    val prevAt = state.lastAtStop
    if (prevAt != null && prevAt < nodes.lastIndex) {
        val prevNode = nodes[prevAt]
        val dFromPrev = geoDistanceMeters(lat, lon, prevNode.lat, prevNode.lon)
        if (dFromPrev > STOP_EXIT_RADIUS_M && (closestIdx != prevAt || dFromPrev > STOP_EXIT_RADIUS_M)) {
            if (prevAt !in state.observedDepartures && (nowMillis - state.lastAtStopMillis) <= 30 * 60_000L) {
                state.observedDepartures[prevAt] = nowMillis
                prevNode.depMillis?.let { state.plannedAtDeparture[prevAt] = it }
            }
            if (prevAt == 0) {
                state.hasLeftStart = true
            }
            state.lastAtStop = null
        }
    }

    if (closestDist <= STOP_ENTER_RADIUS_M) {
        state.visitedStops.add(closestIdx)
        state.lastAtStop = closestIdx
        state.lastAtStopMillis = nowMillis
        if (closestIdx > 0) {
            state.hasLeftStart = true
        }
        if (closestIdx > state.furthestStop) {
            state.furthestStop = closestIdx
        }
        return GpsStopMatch(
            stopPosition = closestIdx,
            isAtStop = true,
            leftStartByGps = state.hasLeftStart
        )
    }

    val polyMatch = matchAlongPolyline(leg.coordinates, nodes, lat, lon)
    val segMatch = polyMatch ?: matchAlongStopSegments(nodes, lat, lon)
    if (segMatch == null || segMatch.second > CORRIDOR_LOCK_RADIUS_M) {
        return null
    }

    val segStartIdx = segMatch.first.coerceIn(0, nodes.lastIndex - 1)
    val nextStopPos = (segStartIdx + 1).coerceIn(1, nodes.lastIndex)
    if (segStartIdx > 0 || closestDist > STOP_EXIT_RADIUS_M) {
        state.hasLeftStart = true
    }
    if (nextStopPos > state.furthestStop) {
        state.furthestStop = nextStopPos
    }

    return GpsStopMatch(
        stopPosition = nextStopPos,
        isAtStop = false,
        leftStartByGps = state.hasLeftStart
    )
}

private fun matchAlongPolyline(
    coords: List<List<Double>>?,
    nodes: List<StopNode>,
    lat: Double,
    lon: Double
): Pair<Int, Float>? {
    if (coords == null || coords.size < 2 || nodes.size < 2) return null

    val cumDist = DoubleArray(coords.size)
    for (i in 1 until coords.size) {
        val prev = coords[i - 1]
        val curr = coords[i]
        cumDist[i] = cumDist[i - 1] + geoDistanceMeters(prev[0], prev[1], curr[0], curr[1]).toDouble()
    }
    if (cumDist.last() < 20.0) return null

    fun projectOnPoly(pLat: Double, pLon: Double): Pair<Double, Float> {
        var bestD = Float.MAX_VALUE
        var bestS = 0.0
        for (i in 0 until coords.lastIndex) {
            val a = coords[i]
            val b = coords[i + 1]
            val proj = projectPointToSegment(pLat, pLon, a[0], a[1], b[0], b[1])
            if (proj.distMeters < bestD) {
                bestD = proj.distMeters
                bestS = cumDist[i] + proj.t * (cumDist[i + 1] - cumDist[i])
            }
        }
        return bestS to bestD
    }

    val (gpsS, gpsDist) = projectOnPoly(lat, lon)
    val stopS = DoubleArray(nodes.size)
    var lastS = 0.0
    for (k in nodes.indices) {
        val n = nodes[k]
        if (n.lat == 0.0 && n.lon == 0.0) {
            stopS[k] = lastS
            continue
        }
        val (s, _) = projectOnPoly(n.lat, n.lon)
        lastS = max(lastS, s)
        stopS[k] = lastS
    }

    for (k in 0 until nodes.lastIndex) {
        if (gpsS <= stopS[k + 1]) {
            return k to gpsDist
        }
    }
    return (nodes.lastIndex - 1) to gpsDist
}

private fun matchAlongStopSegments(
    nodes: List<StopNode>,
    lat: Double,
    lon: Double
): Pair<Int, Float>? {
    var bestSeg = -1
    var bestDist = Float.MAX_VALUE
    for (i in 0 until nodes.lastIndex) {
        val a = nodes[i]
        val b = nodes[i + 1]
        if ((a.lat == 0.0 && a.lon == 0.0) || (b.lat == 0.0 && b.lon == 0.0)) continue
        val proj = projectPointToSegment(lat, lon, a.lat, a.lon, b.lat, b.lon)
        if (proj.distMeters < bestDist) {
            bestDist = proj.distMeters
            bestSeg = i
        }
    }
    return if (bestSeg >= 0) bestSeg to bestDist else null
}

internal fun computeLegScheduleOffsets(
    nodes: List<StopNode>,
    state: LegGpsTrackState
): Map<Int, Long> {
    if (state.observedDepartures.isEmpty() || nodes.size < 2) return emptyMap()

    val latestDepStop = state.observedDepartures.keys.sortedDescending().firstOrNull { k ->
        (state.plannedAtDeparture[k] ?: nodes.getOrNull(k)?.depMillis) != null
    } ?: return emptyMap()

    val plannedDep = state.plannedAtDeparture[latestDepStop] ?: nodes[latestDepStop].depMillis ?: return emptyMap()
    val actualDep = state.observedDepartures[latestDepStop] ?: return emptyMap()
    val rawDelta = actualDep - plannedDep

    if (abs(rawDelta) > LIGHT_OFFSET_MAX_MS) {
        return emptyMap()
    }

    val offsets = mutableMapOf<Int, Long>()
    var prevAdjusted = actualDep
    for (j in (latestDepStop + 1) until nodes.size) {
        val hop = (j - latestDepStop).coerceAtMost(5)
        val divisor = 1L shl hop
        val rawOffset = rawDelta / divisor
        val baseTime = nodes[j].arrMillis ?: nodes[j].depMillis
        if (baseTime != null) {
            var adjusted = baseTime + rawOffset
            if (adjusted < prevAdjusted + 20_000L) {
                adjusted = prevAdjusted + 20_000L
            }
            offsets[j] = adjusted - baseTime
            prevAdjusted = adjusted
        } else {
            offsets[j] = rawOffset
        }
    }
    return offsets
}

internal fun applyScheduleOffsetsToLeg(
    leg: TripLeg,
    offsets: Map<Int, Long>
): TripLeg {
    if (offsets.isEmpty() || leg.type != LegType.TRANSIT) return leg
    val inter = leg.intermediateStops.orEmpty()
    val lastIdx = inter.size + 1

    val newInter = inter.mapIndexed { idx, stop ->
        val stopPos = idx + 1
        val off = offsets[stopPos] ?: 0L
        if (off == 0L) {
            stop
        } else {
            val baseArr = parseIsoTimeMillis(stop.arrivalTime.ifBlank { stop.departureTime })
            val baseDep = parseIsoTimeMillis(stop.departureTime.ifBlank { stop.arrivalTime })
            val adjArr = baseArr?.let { formatMillisLike(stop.arrivalTime.ifBlank { stop.departureTime }, it + off) } ?: stop.arrivalTime
            val adjDep = baseDep?.let { formatMillisLike(stop.departureTime.ifBlank { stop.arrivalTime }, it + off) } ?: stop.departureTime
            val adjDelaySec = stop.delaySec + (off / 1000L).toInt()
            stop.copy(
                arrivalTime = adjArr,
                departureTime = adjDep,
                scheduledArrivalTime = stop.scheduledArrivalTime ?: stop.arrivalTime.takeIf { it.isNotBlank() },
                scheduledDepartureTime = stop.scheduledDepartureTime ?: stop.departureTime.takeIf { it.isNotBlank() },
                delaySec = adjDelaySec,
                isDelayed = stop.isDelayed || adjDelaySec >= 60
            )
        }
    }

    val endOff = offsets[lastIdx] ?: 0L
    val newArr = if (endOff != 0L) {
        parseIsoTimeMillis(leg.arrivalTime)?.let { formatMillisLike(leg.arrivalTime, it + endOff) } ?: leg.arrivalTime
    } else {
        leg.arrivalTime
    }
    val adjArrDelaySec = leg.arrivalDelaySec + (endOff / 1000L).toInt()

    return leg.copy(
        intermediateStops = newInter,
        arrivalTime = newArr,
        scheduledArrivalTime = if (endOff != 0L) (leg.scheduledArrivalTime ?: leg.arrivalTime) else leg.scheduledArrivalTime,
        arrivalDelaySec = adjArrDelaySec,
        isDelayed = leg.isDelayed || adjArrDelaySec >= 60
    )
}

private fun formatCountdown(targetMillis: Long?, nowMillis: Long): String? {
    if (targetMillis == null) return null
    val diffSec = ((targetMillis - nowMillis) / 1000L).toInt()
    return when {
        diffSec <= 0 -> null
        diffSec < 60 -> "Imminent"
        else -> "dans ${(diffSec + 30) / 60} min"
    }
}

fun resolveTrackedNavigationProgress(
    trip: ParsedTrip,
    location: Pair<Float, Float>?,
    gpsFixMillis: Long,
    nowMillis: Long,
    tracker: NavigationSessionTracker,
    exitSide: String? = null,
    platformRef: String? = null,
    platformHints: Map<Int, LegPlatformHint> = emptyMap(),
    gpsFreshWindowMs: Long = 60_000L
): NavigationProgress {
    val legs = trip.legs
    val baseArrivalRaw = legs.lastOrNull()?.arrivalTime ?: trip.arrivalTime
    val baseArrivalMillis = parseIsoTimeMillis(baseArrivalRaw) ?: parseIsoTimeMillis(trip.arrivalTime)
    if (legs.isEmpty()) {
        val remMin = if (baseArrivalMillis != null) max(0, ((baseArrivalMillis - nowMillis) / 60_000L).toInt())
        else max(0, trip.durationSec / 60)
        return NavigationProgress(
            activeLegIndex = 0,
            activeStopPosition = null,
            isAtStop = false,
            isAlightingSoon = false,
            stopsRemaining = null,
            nextActionTitle = "Trajet",
            nextActionSubtitle = null,
            remainingMinutes = remMin,
            arrivalTimeText = formatTimeString(trip.arrivalTime),
            adjustedLegs = emptyList(),
            gpsSynced = location != null
        )
    }

    val freshLocation = if (location != null && (gpsFixMillis == 0L || (nowMillis - gpsFixMillis) <= gpsFreshWindowMs)) location else null
    val gpsIsFresh = freshLocation != null

    val timeLegIndex = legs.indexOfFirst { leg ->
        val arr = parseIsoTimeMillis(leg.arrivalTime)
        arr == null || arr > nowMillis
    }.let { idx -> if (idx == -1) legs.lastIndex else idx }

    var candidateIdx = maxOf(tracker.maxReachedLegIndex, timeLegIndex).coerceIn(0, legs.lastIndex)

    if (freshLocation != null) {
        val lat = freshLocation.first.toDouble()
        val lon = freshLocation.second.toDouble()

        var bestLegIdx = -1
        var bestLegDist = Float.MAX_VALUE
        legs.forEachIndexed { idx, leg ->
            val d = legCorridorDistanceMeters(leg, lat, lon) ?: return@forEachIndexed

            val effectiveD = if (leg.type == LegType.TRANSIT) d else d + 35f
            if (effectiveD < bestLegDist) {
                bestLegDist = effectiveD
                bestLegIdx = idx
            }
        }

        val currentDist = legCorridorDistanceMeters(legs[candidateIdx], lat, lon) ?: Float.MAX_VALUE

        if (bestLegIdx in 0 until candidateIdx &&
            bestLegDist <= 220f &&
            currentDist > 450f
        ) {
            for (resetIdx in (bestLegIdx + 1)..legs.lastIndex) {
                tracker.stateForLeg(resetIdx).reset()
            }
            candidateIdx = bestLegIdx
            tracker.maxReachedLegIndex = bestLegIdx
        } else {

            for (idx in (candidateIdx + 1)..legs.lastIndex) {
                val leg = legs[idx]
                if (leg.type != LegType.TRANSIT) continue
                val dep = parseIsoTimeMillis(leg.departureTime)
                if (dep != null && nowMillis < dep - 180_000L) continue
                val d = legCorridorDistanceMeters(leg, lat, lon)
                if (d != null && d <= 220f) {
                    candidateIdx = idx
                }
            }
        }
    }

    while (candidateIdx < legs.lastIndex &&
        (legs[candidateIdx].type == LegType.WALK || legs[candidateIdx].type == LegType.TRANSFERT)
    ) {
        val cur = legs[candidateIdx]
        val next = legs[candidateIdx + 1]
        val curArr = parseIsoTimeMillis(cur.arrivalTime)
        val nextDep = parseIsoTimeMillis(next.departureTime)
        val arriveThreshold = if (cur.type == LegType.TRANSFERT) 45f else 120f
        val atStationByGps = if (freshLocation != null && (next.fromLat != 0.0 || next.fromLon != 0.0)) {
            geoDistanceMeters(freshLocation.first.toDouble(), freshLocation.second.toDouble(), next.fromLat, next.fromLon) <= arriveThreshold
        } else false

        val shouldAdvance = if (freshLocation != null && next.type == LegType.TRANSIT) {
            if (cur.type == LegType.TRANSFERT) {
                (atStationByGps && (curArr == null || nowMillis >= curArr - 30_000L)) ||
                    (nextDep != null && nowMillis >= nextDep - 30_000L)
            } else {
                atStationByGps ||
                    (curArr != null && nowMillis >= curArr + 120_000L && (legCorridorDistanceMeters(cur, freshLocation.first.toDouble(), freshLocation.second.toDouble()) ?: Float.MAX_VALUE) > 250f)
            }
        } else {
            (curArr != null && nowMillis >= curArr) || (nextDep != null && nowMillis >= nextDep - 60_000L)
        }
        if (shouldAdvance) {
            candidateIdx++
        } else {
            break
        }
    }

    val activeLegIndex = candidateIdx
    if (activeLegIndex > tracker.maxReachedLegIndex) {
        tracker.maxReachedLegIndex = activeLegIndex
    }

    val baseActiveLeg = legs[activeLegIndex]
    val nodes = buildLegStopNodes(baseActiveLeg)
    val legState = tracker.stateForLeg(activeLegIndex)

    val gpsMatch = if (baseActiveLeg.type == LegType.TRANSIT && freshLocation != null) {
        updateAndResolveTransitGps(
            leg = baseActiveLeg,
            nodes = nodes,
            lat = freshLocation.first.toDouble(),
            lon = freshLocation.second.toDouble(),
            nowMillis = nowMillis,
            state = legState
        )
    } else null

    val stopOffsets = if (baseActiveLeg.type == LegType.TRANSIT) {
        computeLegScheduleOffsets(nodes, legState)
    } else emptyMap()

    val adjustedLegs = legs.mapIndexed { idx, leg ->
        if (idx == activeLegIndex && stopOffsets.isNotEmpty()) {
            applyScheduleOffsetsToLeg(leg, stopOffsets)
        } else {
            leg
        }
    }
    val activeLeg = adjustedLegs[activeLegIndex]
    val adjustedNodes = buildLegStopNodes(activeLeg)

    val depMillis = adjustedNodes.firstOrNull()?.depMillis
    val arrMillis = adjustedNodes.lastOrNull()?.arrMillis

    val (activeStopPosition, isAtStop) = if (activeLeg.type == LegType.TRANSIT) {
        if (gpsMatch != null) {
            gpsMatch.stopPosition to gpsMatch.isAtStop
        } else {

            when {
                depMillis != null && nowMillis < depMillis -> 0 to true
                arrMillis != null && nowMillis >= arrMillis -> adjustedNodes.lastIndex to true
                else -> {
                    val intermediateCount = activeLeg.intermediateStops.orEmpty().size
                    val nextIdx = (1..intermediateCount).firstOrNull { pos ->
                        val node = adjustedNodes[pos]
                        val t = node.depMillis ?: node.arrMillis
                        t == null || t > nowMillis
                    }
                    if (nextIdx == null) {
                        adjustedNodes.lastIndex to false
                    } else {
                        val node = adjustedNodes[nextIdx]
                        val dwelling = node.arrMillis != null && node.depMillis != null &&
                            nowMillis >= node.arrMillis && nowMillis <= node.depMillis
                        nextIdx to dwelling
                    }
                }
            }
        }
    } else {
        null to false
    }

    val stopsRemaining: Int? = if (activeLeg.type == LegType.TRANSIT) {
        val pos = activeStopPosition ?: 0
        max(1, adjustedNodes.lastIndex - pos)
    } else null

    val notBoardedYet = if (activeLeg.type == LegType.TRANSIT) {
        when {
            gpsMatch != null -> gpsMatch.stopPosition == 0 && gpsMatch.isAtStop && !gpsMatch.leftStartByGps
            else -> depMillis != null && nowMillis < depMillis
        }
    } else false

    val isAlightingSoon = activeLeg.type == LegType.TRANSIT &&
        !notBoardedYet &&
        stopsRemaining != null && stopsRemaining in 1..2

    val lastStopOffset = stopOffsets[adjustedNodes.lastIndex] ?: 0L
    val adjustedTripArrivalMillis = baseArrivalMillis?.let {
        if (activeLegIndex == legs.lastIndex || (activeLegIndex == legs.lastIndex - 1 && legs.last().type == LegType.WALK)) {
            it + lastStopOffset
        } else {
            it
        }
    }
    val arrivalTimeText = adjustedTripArrivalMillis?.let {
        formatTimeString(formatMillisLike(trip.arrivalTime, it))
    } ?: formatTimeString(trip.arrivalTime)

    val remainingMinutes = if (adjustedTripArrivalMillis != null) {
        max(0, ((adjustedTripArrivalMillis - nowMillis) / 60_000L).toInt())
    } else {
        max(0, trip.durationSec / 60)
    }

    val countdown = formatCountdown(depMillis, nowMillis)
    val targetTransitIdx = if (activeLeg.type == LegType.TRANSIT) {
        activeLegIndex
    } else {
        val rel = adjustedLegs.drop(activeLegIndex + 1).indexOfFirst { it.type == LegType.TRANSIT }
        if (rel >= 0) activeLegIndex + 1 + rel else -1
    }
    val hint = if (targetTransitIdx >= 0) platformHints[targetTransitIdx] else null
    val resolvedExitSide = hint?.exitSide?.takeIf { it == "left" || it == "right" } ?: exitSide
    val resolvedPlatformRef = hint?.ref?.takeIf { it.isNotBlank() } ?: platformRef

    val (title, subtitle) = when (activeLeg.type) {
        LegType.WALK -> {
            val nextTransit = adjustedLegs.drop(activeLegIndex + 1).firstOrNull { it.type == LegType.TRANSIT }
            val t = "Marchez ${distanceToString(activeLeg.distanceKm)} vers ${activeLeg.toStopName}"
            val sub = if (nextTransit != null) {
                val depStr = formatTimeString(nextTransit.departureTime)
                val platform = nextTransit.departurePlatform?.takeIf { it.isNotBlank() }?.let { "Quai $it" }
                val line = nextTransit.route?.shortName?.let { "Ligne $it " } ?: ""
                val arrPlatform = (resolvedPlatformRef ?: nextTransit.arrivalPlatform)?.takeIf { it.isNotBlank() }?.let { "Arrivée Quai $it" }
                val exit = when (resolvedExitSide) {
                    "left" -> "Sortie à gauche"
                    "right" -> "Sortie à droite"
                    else -> null
                }
                listOfNotNull(platform, "${line}à $depStr", arrPlatform, exit).joinToString(" • ")
            } else {
                "Arrivée prévue à $arrivalTimeText"
            }
            t to sub
        }

        LegType.BIKE -> {
            "Pédalez ${distanceToString(activeLeg.distanceKm)} vers ${activeLeg.toStopName}" to
                "Arrivée à ${formatTimeString(activeLeg.arrivalTime)}"
        }

        LegType.TRANSFERT -> {
            val nextTransit = adjustedLegs.drop(activeLegIndex + 1).firstOrNull { it.type == LegType.TRANSIT }
            val t = "Correspondance à ${activeLeg.fromStopName}"
            val sub = if (nextTransit != null) {
                val platform = nextTransit.departurePlatform?.takeIf { it.isNotBlank() }?.let { "Quai $it • " } ?: ""
                val line = nextTransit.route?.shortName?.let { "$it " } ?: ""
                val nextCountdown = formatCountdown(parseIsoTimeMillis(nextTransit.departureTime), nowMillis)
                val arrPlatform = (resolvedPlatformRef ?: nextTransit.arrivalPlatform)?.takeIf { it.isNotBlank() }?.let { "Arrivée Quai $it" }
                listOfNotNull(
                    "${platform}${line}vers ${nextTransit.headsign ?: nextTransit.toStopName}",
                    nextCountdown,
                    arrPlatform
                ).joinToString(" • ")
            } else null
            t to sub
        }

        LegType.TRANSIT -> {
            val lineName = activeLeg.route?.shortName?.takeIf { it.isNotBlank() } ?: "Transport"
            if (notBoardedYet) {
                val depPlatform = activeLeg.departurePlatform?.takeIf { it.isNotBlank() }?.let { "Quai $it" }
                val arrPlatform = (resolvedPlatformRef ?: activeLeg.arrivalPlatform)?.takeIf { it.isNotBlank() }?.let { "Arrivée Quai $it" }
                val exit = when (resolvedExitSide) {
                    "left" -> "Sortie à gauche"
                    "right" -> "Sortie à droite"
                    else -> null
                }
                val trainSize = activeLeg.isLongTrain?.let { if (it) "Train long" else "Train court" }
                val t = "Prenez $lineName vers ${activeLeg.headsign ?: activeLeg.toStopName}"
                val sub = listOfNotNull(
                    countdown ?: "Départ à ${formatTimeString(activeLeg.departureTime)}",
                    depPlatform,
                    trainSize,
                    arrPlatform,
                    exit
                ).joinToString(" • ")
                t to sub
            } else {
                val stopWord = if ((stopsRemaining ?: 1) <= 1) "arrêt" else "arrêts"
                val t = "$lineName • Descendez à ${activeLeg.toStopName}"
                val sub = buildString {
                    append("Dans ${stopsRemaining ?: 1} $stopWord")
                    val arr = formatTimeString(activeLeg.arrivalTime)
                    if (arr.isNotBlank()) append(" • Arrivée $arr")
                    val quai = (resolvedPlatformRef ?: activeLeg.arrivalPlatform)?.takeIf { it.isNotBlank() }
                    if (quai != null) append(" • Quai $quai")
                    when (resolvedExitSide) {
                        "left" -> append(" • Sortie à gauche")
                        "right" -> append(" • Sortie à droite")
                    }
                }
                t to sub
            }
        }
    }

    return NavigationProgress(
        activeLegIndex = activeLegIndex,
        activeStopPosition = activeStopPosition,
        isAtStop = isAtStop,
        isAlightingSoon = isAlightingSoon,
        stopsRemaining = stopsRemaining,
        nextActionTitle = title,
        nextActionSubtitle = subtitle,
        remainingMinutes = remainingMinutes,
        arrivalTimeText = arrivalTimeText,
        adjustedLegs = adjustedLegs,
        gpsSynced = gpsIsFresh
    )
}
