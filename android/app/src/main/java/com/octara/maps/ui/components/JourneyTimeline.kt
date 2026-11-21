package com.octara.maps.ui.components

import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Accessible
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.automirrored.filled.DirectionsBike
import androidx.compose.material.icons.automirrored.filled.DirectionsWalk
import androidx.compose.material.icons.filled.Elevator
import androidx.compose.material.icons.filled.Escalator
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.SensorDoor
import androidx.compose.material.icons.filled.Stairs
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.draw.rotate
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.octara.maps.types.Accessibility
import com.octara.maps.types.LegType
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType
import com.octara.maps.types.TrainPosition
import com.octara.maps.types.TripLeg
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.LocalTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.util.Locale

enum class LegNavigationState {
    PAST,
    ACTIVE,
    UPCOMING
}

private enum class TransitStopMarkerState {
    NORMAL,
    PASSED,
    AT_STOP,
    NEXT_STOP
}

private val ActiveStopYellow = Color(0xFFFFD600)

private fun DrawScope.drawTimelineStopDot(
    state: TransitStopMarkerState,
    center: Offset,
    baseRadius: Float,
    blinkAlpha: Float
) {
    when (state) {
        TransitStopMarkerState.AT_STOP -> {
            drawCircle(
                color = ActiveStopYellow,
                radius = baseRadius * 1.25f,
                center = center
            )
        }

        TransitStopMarkerState.NEXT_STOP -> {
            drawCircle(
                color = Color.White,
                radius = baseRadius,
                center = center
            )
            drawCircle(
                color = ActiveStopYellow.copy(alpha = blinkAlpha),
                radius = baseRadius * 1.25f,
                center = center
            )
        }

        TransitStopMarkerState.PASSED -> {
            drawCircle(
                color = Color.White.copy(alpha = 0.85f),
                radius = baseRadius,
                center = center
            )
        }

        TransitStopMarkerState.NORMAL -> {
            drawCircle(
                color = Color.White,
                radius = baseRadius,
                center = center
            )
        }
    }
}

private fun LegNavigationState?.contentAlpha(): Float = when (this) {
    LegNavigationState.PAST -> 0.60f
    LegNavigationState.UPCOMING, LegNavigationState.ACTIVE, null -> 1f
}

private fun Color.forNavigationState(state: LegNavigationState?): Color = when (state) {
    LegNavigationState.PAST -> lerp(this, Color(0xFF9E9E9E), 0.45f)
    LegNavigationState.UPCOMING, LegNavigationState.ACTIVE, null -> this
}

private fun parseTimeMillisOrNull(time: String?): Long? {
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

private fun formatRemainingStopTime(
    time: String?,
    nowMillis: Long?,
    isAtThisStop: Boolean
): String? {
    if (isAtThisStop) return "0 min"
    if (nowMillis == null) return null
    val targetMillis = parseTimeMillisOrNull(time) ?: return null
    val diffSec = ((targetMillis - nowMillis) / 1000L).toInt()
    return when {
        diffSec <= 0 -> "0 min"
        diffSec < 60 -> "< 1 min"
        else -> "${(diffSec + 30) / 60} min"
    }
}

@Composable
fun JourneyLegItem(
    leg: TripLeg,
    navigationState: LegNavigationState? = null,
    activeStopPosition: Int? = null,
    isAtStop: Boolean = false,
    isAlightingSoon: Boolean = false,
    nowMillis: Long? = null,
    prevLeg: TripLeg? = null,
    nextLeg: TripLeg? = null,
    isFirstWalk: Boolean = false,
    resyncKey: Int = 0,
    externalArrivalPlatformInfo: com.octara.maps.utils.Api.PlatformInfo? = null
) {
    val alpha = navigationState.contentAlpha()
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .graphicsLayer { this.alpha = alpha }
    ) {
        when (leg.type) {
            LegType.TRANSIT -> TransitLegItem(
                leg = leg,
                navigationState = navigationState,
                activeStopPosition = activeStopPosition,
                isAtStop = isAtStop,
                isAlightingSoon = isAlightingSoon,
                nowMillis = nowMillis,
                resyncKey = resyncKey,
                externalArrivalPlatformInfo = externalArrivalPlatformInfo
            )

            LegType.BIKE -> BikingLegItem(leg = leg, navigationState = navigationState)
            LegType.TRANSFERT -> TransfertLegItem(
                leg = leg,
                navigationState = navigationState,
                prevLeg = prevLeg,
                nextLeg = nextLeg,
                resyncKey = resyncKey
            )
            else -> WalkingLegItem(
                leg = leg,
                navigationState = navigationState,
                prevLeg = prevLeg,
                nextLeg = nextLeg,
                isFirstWalk = isFirstWalk,
                resyncKey = resyncKey
            )
        }
    }
}

fun distanceToString(distanceKm: Double): String {
    return if (distanceKm < 1) {
        "${(distanceKm * 1000).toInt()} m"
    } else {
        String.format(Locale.getDefault(), "%.1f km", distanceKm)
    }
}

fun formatTimeString(time: String?): String {
    if (time.isNullOrBlank()) return ""
    val match = Regex("(?:^|T|\\s)(\\d{1,2}):(\\d{2})").find(time)
    return match?.let {
        val h = it.groupValues[1].toInt()
        val m = it.groupValues[2]
        "${h}h$m"
    } ?: ""
}

@Composable
private fun rememberIndoorTransferGuide(
    leg: TripLeg,
    prevLeg: TripLeg?,
    nextLeg: TripLeg?,
    enabled: Boolean,
    resyncKey: Int = 0
): com.octara.maps.utils.Api.IndoorRouteResult? {
    val fromLat = leg.fromLat.takeIf { it != 0.0 } ?: (prevLeg?.toLat ?: 0.0)
    val fromLon = leg.fromLon.takeIf { it != 0.0 } ?: (prevLeg?.toLon ?: 0.0)
    val toLat = leg.toLat.takeIf { it != 0.0 } ?: (nextLeg?.fromLat ?: 0.0)
    val toLon = leg.toLon.takeIf { it != 0.0 } ?: (nextLeg?.fromLon ?: 0.0)

    val fromIsMetroOrTram = prevLeg?.route?.routeType == RouteType.METRO || prevLeg?.route?.routeType == RouteType.TRAM
    val toIsMetroOrTram = nextLeg?.route?.routeType == RouteType.METRO || nextLeg?.route?.routeType == RouteType.TRAM
    val fromRef = prevLeg?.arrivalPlatform?.takeIf { it.isNotBlank() } ?: leg.departurePlatform
    val toRef = nextLeg?.departurePlatform?.takeIf { it.isNotBlank() }

    var guide by remember(leg.fromSID, leg.toSID, fromLat, fromLon, toLat, toLon, enabled) {
        mutableStateOf<com.octara.maps.utils.Api.IndoorRouteResult?>(null)
    }

    LaunchedEffect(enabled, leg.fromSID, leg.toSID, fromLat, fromLon, toLat, toLon, fromRef, toRef, resyncKey) {
        if (!enabled) {
            guide = null
            return@LaunchedEffect
        }
        val api = com.octara.maps.utils.Api()
        for (waitMs in listOf(0L, 2_000L, 5_000L)) {
            if (waitMs > 0L) kotlinx.coroutines.delay(waitMs)
            val res = api.getTransferGuide(
                fromLat = fromLat,
                fromLon = fromLon,
                toLat = toLat,
                toLon = toLon,
                fromRef = fromRef,
                toRef = toRef,
                fromNearest = fromIsMetroOrTram,
                toNearest = toIsMetroOrTram,
                wheelchair = false
            )
            if (res != null) {
                guide = res
                break
            }
        }
    }
    return guide
}

@Composable
fun TransfertLegItem(
    leg: TripLeg,
    navigationState: LegNavigationState? = null,
    prevLeg: TripLeg? = null,
    nextLeg: TripLeg? = null,
    resyncKey: Int = 0
) {
    val baseColor = if (navigationState == LegNavigationState.ACTIVE) Color(0xFF1E88E5) else Color(0xFF757575)
    val tColor = baseColor.forNavigationState(navigationState)
    val guide = rememberIndoorTransferGuide(
        leg = leg,
        prevLeg = prevLeg,
        nextLeg = nextLeg,
        enabled = prevLeg != null && nextLeg != null,
        resyncKey = resyncKey
    )
    Column(modifier = Modifier.fillMaxWidth()) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp)
        ) {
            Box(
                modifier = Modifier.width(16.dp),
                contentAlignment = Alignment.Center
            ) {
                Canvas(
                    modifier = Modifier
                        .width(16.dp)
                        .height(55.dp)
                ) {
                    val strokeWidth = 6.dp.toPx()
                    val centerX = size.width / 2f
                    val arrowWidth = 7.dp.toPx()
                    val arrowHeadHeight = 7.dp.toPx()

                    val topPath = Path().apply {
                        moveTo(centerX - arrowWidth, arrowHeadHeight)
                        lineTo(centerX, 0f)
                        lineTo(centerX + arrowWidth, arrowHeadHeight)
                    }
                    drawPath(
                        path = topPath,
                        color = tColor,
                        style = Stroke(
                            width = strokeWidth,
                            cap = StrokeCap.Round,
                            join = StrokeJoin.Round
                        )
                    )

                    val bottomPath = Path().apply {
                        moveTo(centerX - arrowWidth, size.height - arrowHeadHeight)
                        lineTo(centerX, size.height)
                        lineTo(centerX + arrowWidth, size.height - arrowHeadHeight)
                    }
                    drawPath(
                        path = bottomPath,
                        color = tColor,
                        style = Stroke(
                            width = strokeWidth,
                            cap = StrokeCap.Round,
                            join = StrokeJoin.Round
                        )
                    )

                    drawLine(
                        color = tColor,
                        start = Offset(centerX, 0f),
                        end = Offset(centerX, size.height),
                        strokeWidth = strokeWidth,
                        cap = StrokeCap.Round
                    )
                }
            }

            Text(
                text = "Correspondance",
                fontSize = 20.sp,
                fontWeight = if (navigationState == LegNavigationState.ACTIVE) FontWeight.SemiBold else FontWeight.Medium,
                color = tColor,
                modifier = Modifier.padding(start = 12.dp)
            )
        }
        IndoorTransferStepsSection(
            leg = leg,
            guide = guide,
            navigationState = navigationState,
            accentColor = tColor
        )
    }
}

@Composable
fun WalkingLegItem(
    leg: TripLeg,
    navigationState: LegNavigationState? = null,
    prevLeg: TripLeg? = null,
    nextLeg: TripLeg? = null,
    isFirstWalk: Boolean = false,
    resyncKey: Int = 0
) {
    val baseColor = if (navigationState == LegNavigationState.ACTIVE) Color(0xFF1E88E5) else Color(0xFF757575)
    val tColor = baseColor.forNavigationState(navigationState)
    val isInsideStationWalk = !isFirstWalk && prevLeg != null && nextLeg != null
    val guide = rememberIndoorTransferGuide(
        leg = leg,
        prevLeg = prevLeg,
        nextLeg = nextLeg,
        enabled = isInsideStationWalk,
        resyncKey = resyncKey
    )
    val displayDistText = if (leg.distanceKm <= 0.0 && (guide?.total_distance_m ?: 0) > 0) {
        "${guide!!.total_distance_m} m"
    } else {
        distanceToString(leg.distanceKm)
    }
    Column(modifier = Modifier.fillMaxWidth()) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 8.dp)
        ) {
            Box(
                modifier = Modifier.width(16.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                    modifier = Modifier.height(55.dp)
                ) {
                    repeat(4) {
                        Box(
                            modifier = Modifier
                                .width(6.dp)
                                .height(10.dp)
                                .background(tColor, RoundedCornerShape(3.dp))
                        )
                    }
                }
            }

            Icon(
                imageVector = Icons.AutoMirrored.Filled.DirectionsWalk,
                contentDescription = "Marcher",
                tint = tColor,
                modifier = Modifier.size(26.dp)
            )
            Text(
                text = "Marcher durant $displayDistText",
                fontSize = 20.sp,
                fontWeight = if (navigationState == LegNavigationState.ACTIVE) FontWeight.SemiBold else FontWeight.Medium,
                color = tColor,
                modifier = Modifier.padding(start = 12.dp)
            )
        }
        IndoorTransferStepsSection(
            leg = leg,
            guide = guide,
            navigationState = navigationState,
            accentColor = tColor
        )
    }
}

@Composable
fun IndoorTransferStepsSection(
    leg: TripLeg,
    guide: com.octara.maps.utils.Api.IndoorRouteResult?,
    navigationState: LegNavigationState?,
    accentColor: Color
) {
    var isExpanded by remember(leg.fromSID, leg.toSID) {
        mutableStateOf(navigationState == LegNavigationState.ACTIVE)
    }
    var manualStepSkip by remember(leg.fromSID, leg.toSID) { mutableStateOf(0) }

    LaunchedEffect(navigationState) {
        if (navigationState == LegNavigationState.ACTIVE) {
            isExpanded = true
        }
    }

    val steps = guide?.steps.orEmpty()
    if (guide?.found != true || steps.isEmpty()) return

    val trackerState by com.octara.maps.utils.IndoorStepTracker.state.collectAsState()
    val isActive = navigationState == LegNavigationState.ACTIVE

    val activeStepIdx = if (isActive) {
        var remaining = trackerState.distanceWalkedM
        var idx = 0
        for (i in steps.indices) {
            val d = steps[i].distance_m.toFloat().coerceAtLeast(1f)
            if (remaining < d) {
                idx = i
                break
            }
            remaining -= d
            idx = i
        }
        (idx + manualStepSkip).coerceIn(0, steps.lastIndex)
    } else {
        -1
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 40.dp, end = 12.dp, bottom = 6.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .clickable { isExpanded = !isExpanded }
                .padding(vertical = 4.dp, horizontal = 4.dp)
        ) {
            Text(
                text = if (isExpanded) {
                    "Masquer le guidage (${steps.size} étapes • ${guide?.total_distance_m ?: 0} m)"
                } else {
                    "Guidage détaillé (${steps.size} étapes • ${guide?.total_distance_m ?: 0} m)"
                },
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = accentColor
            )
            Icon(
                imageVector = if (isExpanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
                contentDescription = null,
                tint = accentColor,
                modifier = Modifier.size(18.dp)
            )
        }

        if (isExpanded) {
            steps.forEachIndexed { index, step ->
                val isCurrentStep = isActive && index == activeStepIdx
                val stepIcon = when (step.kind) {
                    "stairs" -> Icons.Default.Stairs
                    "escalator" -> Icons.Default.Escalator
                    "elevator" -> Icons.Default.Elevator
                    "gate" -> Icons.Default.SensorDoor
                    else -> Icons.AutoMirrored.Filled.DirectionsWalk
                }

                Surface(
                    shape = RoundedCornerShape(10.dp),
                    color = if (isCurrentStep) {
                        accentColor.copy(alpha = 0.14f)
                    } else {
                        MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f)
                    },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp)
                    ) {
                        Icon(
                            imageVector = stepIcon,
                            contentDescription = step.kind,
                            tint = if (isCurrentStep) accentColor else MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.size(20.dp)
                        )
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = step.instruction,
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = if (isCurrentStep) FontWeight.Bold else FontWeight.Medium,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                            val levelTag = when {
                                !step.from_level.isNullOrBlank() && !step.to_level.isNullOrBlank() && step.from_level != step.to_level ->
                                    "Niv. ${step.from_level} → ${step.to_level}"
                                !step.to_level.isNullOrBlank() -> "Niv. ${step.to_level}"
                                else -> null
                            }
                            val meta = listOfNotNull(
                                "${step.distance_m} m",
                                levelTag,
                                if (isCurrentStep && trackerState.steps > 0) "${trackerState.steps} pas" else null
                            ).joinToString(" • ")
                            Text(
                                text = meta,
                                style = MaterialTheme.typography.labelSmall,
                                color = if (isCurrentStep) accentColor else MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                        if (isCurrentStep) {
                            Icon(
                                imageVector = Icons.Default.Navigation,
                                contentDescription = "Direction",
                                tint = accentColor,
                                modifier = Modifier
                                    .size(20.dp)
                                    .rotate(step.bearing_deg.toFloat() - trackerState.headingDeg)
                            )
                            if (index < steps.lastIndex) {
                                Text(
                                    text = "Suivant",
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    color = accentColor,
                                    modifier = Modifier
                                        .clip(RoundedCornerShape(6.dp))
                                        .clickable { manualStepSkip++ }
                                        .padding(horizontal = 6.dp, vertical = 4.dp)
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun BikingLegItem(
    leg: TripLeg,
    navigationState: LegNavigationState? = null
) {
    val baseColor = if (navigationState == LegNavigationState.ACTIVE) Color(0xFF1E88E5) else Color(0xFF757575)
    val tColor = baseColor.forNavigationState(navigationState)
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
    ) {
        Box(
            modifier = Modifier.width(16.dp),
            contentAlignment = Alignment.Center
        ) {
            Column(
                verticalArrangement = Arrangement.spacedBy(4.dp),
                modifier = Modifier.height(55.dp)
            ) {
                repeat(4) {
                    Box(
                        modifier = Modifier
                            .width(6.dp)
                            .height(10.dp)
                            .background(tColor, RoundedCornerShape(3.dp))
                    )
                }
            }
        }

        Icon(
            imageVector = Icons.AutoMirrored.Filled.DirectionsBike,
            contentDescription = "Vélo",
            tint = tColor,
            modifier = Modifier.size(26.dp)
        )
        Text(
            text = "Pédaler durant ${distanceToString(leg.distanceKm)}",
            fontSize = 20.sp,
            fontWeight = if (navigationState == LegNavigationState.ACTIVE) FontWeight.SemiBold else FontWeight.Medium,
            color = tColor,
            modifier = Modifier.padding(start = 12.dp)
        )
    }
}

@Composable
fun TransitLegItem(
    leg: TripLeg,
    navigationState: LegNavigationState? = null,
    activeStopPosition: Int? = null,
    isAtStop: Boolean = false,
    isAlightingSoon: Boolean = false,
    nowMillis: Long? = null,
    resyncKey: Int = 0,
    externalArrivalPlatformInfo: com.octara.maps.utils.Api.PlatformInfo? = null
) {
    val route = leg.route
    val rgb = route?.color?.removePrefix("#")?.toLongOrNull(16)?.toInt()
        ?: 0x0071BD
    val rawColor = Color(rgb or 0xFF000000.toInt())
    val tColor = rawColor.forNavigationState(navigationState)

    val intermediateStops = leg.intermediateStops ?: emptyList()

    var localArrivalPlatformInfo by remember(leg.toSID, leg.toLat, leg.toLon) {
        mutableStateOf<com.octara.maps.utils.Api.PlatformInfo?>(null)
    }
    LaunchedEffect(leg.toSID, leg.route?.rid, leg.toLat, leg.toLon, resyncKey, externalArrivalPlatformInfo != null) {
        if (externalArrivalPlatformInfo != null) return@LaunchedEffect
        val api = com.octara.maps.utils.Api()
        while (true) {
            val info = api.resolveLiveArrivalPlatform(leg)
            if (info != null) {
                localArrivalPlatformInfo = info
                kotlinx.coroutines.delay(25_000L)
            } else {
                kotlinx.coroutines.delay(5_000L)
            }
        }
    }
    val arrivalPlatformInfo = externalArrivalPlatformInfo ?: localArrivalPlatformInfo
    var isIntermediateExpanded by remember {
        mutableStateOf(navigationState == LegNavigationState.ACTIVE)
    }

    LaunchedEffect(navigationState) {
        if (navigationState == LegNavigationState.ACTIVE) {
            isIntermediateExpanded = true
        } else if (navigationState == LegNavigationState.PAST) {
            isIntermediateExpanded = false
        }
    }

    val blinkAlpha = if (navigationState == LegNavigationState.ACTIVE && !isAtStop && activeStopPosition != null) {
        val infiniteTransition = rememberInfiniteTransition(label = "stopBlink")
        val anim by infiniteTransition.animateFloat(
            initialValue = 1f,
            targetValue = 0.15f,
            animationSpec = infiniteRepeatable(
                animation = tween(durationMillis = 600, easing = FastOutSlowInEasing),
                repeatMode = RepeatMode.Reverse
            ),
            label = "blinkAlpha"
        )
        anim
    } else {
        1f
    }

    fun markerStateFor(pos: Int): TransitStopMarkerState {
        if (navigationState != LegNavigationState.ACTIVE || activeStopPosition == null) {
            return TransitStopMarkerState.NORMAL
        }
        return when {
            pos < activeStopPosition -> TransitStopMarkerState.PASSED
            pos == activeStopPosition -> if (isAtStop) TransitStopMarkerState.AT_STOP else TransitStopMarkerState.NEXT_STOP
            else -> TransitStopMarkerState.NORMAL
        }
    }

    fun shouldShowRemainingTime(pos: Int): Boolean {
        if (navigationState != LegNavigationState.ACTIVE || activeStopPosition == null || nowMillis == null) {
            return false
        }
        return pos in activeStopPosition until (activeStopPosition + 3)
    }

    val headerBarColor = tColor

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
    ) {
        val barWidth = 16.dp
        val iconHeight = 22.dp
        val iconLeftOffset = (barWidth - iconHeight) / 2

        if (route != null) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 6.dp)
            ) {
                route.vehiculeImageUrl?.let { url ->
                    val vehicleCardHeight = 34.dp
                    Box(
                        modifier = Modifier
                            .offset(x = iconLeftOffset)
                            .height(vehicleCardHeight)
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(topEnd = 6.dp, bottomEnd = 6.dp)),
                        contentAlignment = Alignment.CenterStart
                    ) {
                        AsyncImage(
                            model = url,
                            contentDescription = "Véhicule",
                            modifier = Modifier
                                .fillMaxHeight()
                                .graphicsLayer { alpha = 0.99f }
                                .drawWithContent {
                                    drawContent()
                                    drawRect(
                                        brush = Brush.horizontalGradient(
                                            0.0f to Color.Transparent,
                                            0.35f to Color.Black
                                        ),
                                        blendMode = BlendMode.DstIn
                                    )
                                },
                            contentScale = ContentScale.FillHeight,
                            alignment = Alignment.CenterEnd
                        )
                    }
                }
                Box(modifier = Modifier.offset(x = iconLeftOffset)) {
                    RouteIcon(
                        route = Route(
                            id = route.rid,
                            picto = route.picto,
                            color = route.color,
                            tcolor = route.textColor,
                            short_name = route.shortName,
                            long_name = route.longName,
                            type = route.routeType
                        ),
                        modifier = Modifier.height(iconHeight)
                    )
                }
            }
        }

        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .height(IntrinsicSize.Min)
        ) {
            Canvas(
                modifier = Modifier
                    .width(16.dp)
                    .fillMaxHeight()
            ) {
                val pillWidth = size.width
                val radius = pillWidth / 2f
                val holeRadius = radius * 0.52f
                val centerY = size.height / 2f

                val topY = (centerY - radius).coerceAtLeast(0f)
                val bottomY = size.height

                val path = Path().apply {
                    addRoundRect(
                        RoundRect(
                            left = 0f,
                            top = topY,
                            right = pillWidth,
                            bottom = bottomY,
                            topLeftCornerRadius = CornerRadius(radius, radius),
                            topRightCornerRadius = CornerRadius(radius, radius),
                            bottomRightCornerRadius = CornerRadius.Zero,
                            bottomLeftCornerRadius = CornerRadius.Zero
                        )
                    )
                }
                drawPath(path, color = headerBarColor)

                drawTimelineStopDot(
                    state = markerStateFor(0),
                    center = Offset(x = radius, y = centerY),
                    baseRadius = holeRadius,
                    blinkAlpha = blinkAlpha
                )
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 12.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = leg.fromStopName,
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Medium,
                    color = headerBarColor,
                    modifier = Modifier.weight(1f, fill = false)
                )
                val timeStr = formatTimeString(leg.departureTime)
                val schedDepStr = formatTimeString(leg.scheduledDepartureTime)
                val isDepDelayed =
                    leg.departureDelaySec >= 60 || (leg.isDelayed && schedDepStr.isNotBlank() && schedDepStr != timeStr)
                if (timeStr.isNotBlank()) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(start = 8.dp)
                    ) {
                        if (isDepDelayed && schedDepStr.isNotBlank() && schedDepStr != timeStr) {
                            Text(
                                text = schedDepStr,
                                fontSize = 13.sp,
                                style = MaterialTheme.typography.bodySmall.copy(textDecoration = TextDecoration.LineThrough),
                                color = Color(0xFF888888)
                            )
                        }
                        Text(
                            text = timeStr,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            color = if (isDepDelayed) Color(0xFFD84315) else Color(0xFF555555)
                        )
                    }
                }
            }
        }

        if (route?.longName != null) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min)
            ) {
                Box(
                    modifier = Modifier
                        .width(16.dp)
                        .fillMaxHeight()
                        .background(headerBarColor)
                )

                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .padding(start = 12.dp, top = 2.dp, bottom = 4.dp)
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        text = leg.headsign ?: route.longName,
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Normal,
                        color = headerBarColor
                    )
                }
            }
        }

        val hasExtraVehicleInfo = !leg.departurePlatform.isNullOrBlank() || leg.isLongTrain != null
        if (hasExtraVehicleInfo) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min)
            ) {
                Box(
                    modifier = Modifier
                        .width(16.dp)
                        .fillMaxHeight()
                        .background(headerBarColor)
                )

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 12.dp, top = 2.dp, bottom = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    if (!leg.departurePlatform.isNullOrBlank()) {
                        Surface(
                            shape = RoundedCornerShape(6.dp),
                            color = tColor.copy(alpha = 0.15f)
                        ) {
                            Text(
                                text = "Quai ${leg.departurePlatform}",
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold,
                                color = tColor,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                    if (leg.isLongTrain != null) {
                        Surface(
                            shape = RoundedCornerShape(6.dp),
                            color = MaterialTheme.colorScheme.surfaceVariant
                        ) {
                            Text(
                                text = if (leg.isLongTrain) "Train long" else "Train court",
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Medium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                }
            }
        }

        val hasAccessibilityInfo = leg.wheelchair != Accessibility.UNKNOWN || leg.bikesAllowed != Accessibility.UNKNOWN
        if (hasAccessibilityInfo) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min)
            ) {
                Box(
                    modifier = Modifier
                        .width(16.dp)
                        .fillMaxHeight()
                        .background(headerBarColor)
                )

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 12.dp, top = 2.dp, bottom = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    if (leg.wheelchair == Accessibility.YES) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.Accessible,
                            contentDescription = "Accessible aux fauteuils roulants",
                            modifier = Modifier.size(20.dp),
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    } else if (leg.wheelchair == Accessibility.NO) {
                        Box(modifier = Modifier.size(20.dp), contentAlignment = Alignment.Center) {
                            Icon(
                                imageVector = Icons.AutoMirrored.Filled.Accessible,
                                contentDescription = "Non accessible aux fauteuils roulants",
                                modifier = Modifier.size(20.dp),
                                tint = MaterialTheme.colorScheme.error.copy(alpha = 0.75f)
                            )
                            val strokeColor = MaterialTheme.colorScheme.error
                            Canvas(modifier = Modifier.fillMaxSize()) {
                                drawLine(
                                    color = strokeColor,
                                    start = Offset(0f, 0f),
                                    end = Offset(size.width, size.height),
                                    strokeWidth = 2.5.dp.toPx()
                                )
                            }
                        }
                    }

                    if (leg.bikesAllowed == Accessibility.YES) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.DirectionsBike,
                            contentDescription = "Vélos autorisés",
                            modifier = Modifier.size(20.dp),
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    } else if (leg.bikesAllowed == Accessibility.NO) {
                        Box(modifier = Modifier.size(20.dp), contentAlignment = Alignment.Center) {
                            Icon(
                                imageVector = Icons.AutoMirrored.Filled.DirectionsBike,
                                contentDescription = "Vélos non autorisés",
                                modifier = Modifier.size(20.dp),
                                tint = MaterialTheme.colorScheme.error.copy(alpha = 0.75f)
                            )
                            val strokeColor = MaterialTheme.colorScheme.error
                            Canvas(modifier = Modifier.fillMaxSize()) {
                                drawLine(
                                    color = strokeColor,
                                    start = Offset(0f, 0f),
                                    end = Offset(size.width, size.height),
                                    strokeWidth = 2.5.dp.toPx()
                                )
                            }
                        }
                    }
                }
            }
        }

        val posInTrain = leg.positionInTrain
        if (posInTrain != null && posInTrain.nbWagons != 0) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min)
            ) {
                Box(
                    modifier = Modifier
                        .width(16.dp)
                        .fillMaxHeight()
                        .background(headerBarColor)
                )

                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 12.dp, top = 4.dp, bottom = 6.dp)
                ) {
                    Text(
                        text = when (posInTrain.posAvg) {
                            TrainPosition.FRONT -> "Montez à l'avant du train (wagon ${posInTrain.wagon})"
                            TrainPosition.MIDDLE -> "Montez au milieu du train (wagon ${posInTrain.wagon})"
                            TrainPosition.BACK -> "Montez à l'arrière du train (wagon ${posInTrain.wagon})"
                            else -> "Montez dans le wagon ${posInTrain.wagon}"
                        },
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Normal,
                        color = tColor
                    )

                    Spacer(modifier = Modifier.height(6.dp))

                    val highlightedIndex = when (posInTrain.posAvg) {
                        TrainPosition.FRONT -> 2
                        TrainPosition.MIDDLE -> 1
                        TrainPosition.BACK -> 0
                        TrainPosition.UNKNOWN -> -1
                    }

                    Row(
                        modifier = Modifier.height(20.dp),
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Canvas(modifier = Modifier.width(42.dp).fillMaxHeight()) {
                            val isHighlighted = highlightedIndex == 0
                            val color = if (isHighlighted) tColor else tColor.copy(alpha = 0.22f)
                            val w = size.width
                            val h = size.height
                            val r = 2.5.dp.toPx()

                            val path = Path().apply {
                                moveTo(0f, h - r)
                                cubicTo(0f, h * 0.45f, w * 0.25f, 0f, w * 0.6f, 0f)
                                lineTo(w - r, 0f)
                                quadraticTo(w, 0f, w, r)
                                lineTo(w, h - r)
                                quadraticTo(w, h, w - r, h)
                                lineTo(r, h)
                                quadraticTo(0f, h, 0f, h - r)
                                close()
                            }
                            drawPath(path, color = color)
                        }

                        Canvas(modifier = Modifier.width(48.dp).fillMaxHeight()) {
                            val isHighlighted = highlightedIndex == 1
                            val color = if (isHighlighted) tColor else tColor.copy(alpha = 0.22f)
                            val r = 3.dp.toPx()
                            drawRoundRect(
                                color = color,
                                cornerRadius = CornerRadius(r, r)
                            )
                        }

                        Canvas(modifier = Modifier.width(42.dp).fillMaxHeight()) {
                            val isHighlighted = highlightedIndex == 2
                            val color = if (isHighlighted) tColor else tColor.copy(alpha = 0.22f)
                            val w = size.width
                            val h = size.height
                            val r = 2.5.dp.toPx()

                            val path = Path().apply {
                                moveTo(0f, r)
                                quadraticTo(0f, 0f, r, 0f)
                                lineTo(w * 0.4f, 0f)
                                cubicTo(w * 0.75f, 0f, w, h * 0.45f, w, h - r)
                                quadraticTo(w, h, w - r, h)
                                lineTo(r, h)
                                quadraticTo(0f, h, 0f, h - r)
                                close()
                            }
                            drawPath(path, color = color)
                        }
                    }
                }
            }
        }

        if (intermediateStops.isNotEmpty()) {
            Box(
                modifier = Modifier
                    .width(16.dp)
                    .height(14.dp)
                    .background(headerBarColor)
            )

            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(IntrinsicSize.Min)
                    .clickable { isIntermediateExpanded = !isIntermediateExpanded }
            ) {
                Canvas(
                    modifier = Modifier
                        .width(16.dp)
                        .fillMaxHeight()
                ) {
                    val pillWidth = size.width
                    val radius = pillWidth / 2f
                    val holeRadius = radius * 0.52f
                    val centerY = size.height / 2f

                    drawRect(
                        color = headerBarColor,
                        topLeft = Offset(0f, 0f),
                        size = Size(pillWidth, size.height)
                    )

                    if (!isIntermediateExpanded) {
                        drawCircle(
                            color = Color.White,
                            radius = holeRadius,
                            center = Offset(x = radius, y = centerY)
                        )
                    }
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 12.dp, top = 6.dp, bottom = 6.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = if (isIntermediateExpanded) {
                            "Masquer les arrêts intermédiaires (${intermediateStops.size})"
                        } else {
                            "Voir les arrêts intermédiaires (${intermediateStops.size})"
                        },
                        fontSize = 15.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = tColor,
                        modifier = Modifier.weight(1f, fill = false)
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Icon(
                        imageVector = if (isIntermediateExpanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
                        contentDescription = null,
                        modifier = Modifier.size(20.dp),
                        tint = tColor
                    )
                }
            }

            if (isIntermediateExpanded) {
                for (idx in intermediateStops.indices) {
                    val stop = intermediateStops[idx]
                    val isBlocked = stop.isBlocked
                    val stopMarkerState = markerStateFor(idx + 1)
                    val isStopPassed = stopMarkerState == TransitStopMarkerState.PASSED
                    val isStopHighlighted = stopMarkerState == TransitStopMarkerState.AT_STOP ||
                            stopMarkerState == TransitStopMarkerState.NEXT_STOP

                    val isFirstBlocked = isBlocked && (idx == 0 || !intermediateStops[idx - 1].isBlocked)

                    if (isFirstBlocked) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(28.dp)
                        ) {
                            Box(
                                modifier = Modifier
                                    .width(44.dp)
                                    .fillMaxHeight(),
                                contentAlignment = Alignment.CenterStart
                            ) {
                                Canvas(modifier = Modifier.fillMaxSize()) {
                                    val barW = 16.dp.toPx()
                                    drawRect(
                                        color = tColor.copy(alpha = 0.25f),
                                        topLeft = Offset(0f, 0f),
                                        size = Size(barW, size.height)
                                    )
                                    val detourX = 36.dp.toPx()
                                    val path = Path().apply {
                                        moveTo(barW / 2f, 0f)
                                        cubicTo(
                                            barW / 2f, size.height * 0.5f,
                                            detourX, size.height * 0.5f,
                                            detourX, size.height
                                        )
                                    }
                                    drawPath(
                                        path = path,
                                        color = tColor,
                                        style = Stroke(
                                            width = barW,
                                            cap = StrokeCap.Round
                                        )
                                    )
                                }
                            }
                        }
                    }

                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(16.dp)
                    ) {
                        Canvas(
                            modifier = Modifier
                                .width(if (isBlocked) 44.dp else 16.dp)
                                .fillMaxHeight()
                        ) {
                            val barW = 16.dp.toPx()
                            if (isBlocked) {
                                drawRect(
                                    color = tColor.copy(alpha = 0.25f),
                                    topLeft = Offset(0f, 0f),
                                    size = Size(barW, size.height)
                                )

                                val detourX = 36.dp.toPx()
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(detourX - (barW / 2f), 0f),
                                    size = Size(barW, size.height)
                                )
                            } else {
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(0f, 0f),
                                    size = Size(barW, size.height)
                                )
                            }
                        }
                    }

                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(IntrinsicSize.Min)
                    ) {
                        Canvas(
                            modifier = Modifier
                                .width(if (isBlocked) 44.dp else 16.dp)
                                .fillMaxHeight()
                        ) {
                            val barW = 16.dp.toPx()
                            val centerY = size.height / 2f

                            if (isBlocked) {
                                drawRect(
                                    color = tColor.copy(alpha = 0.25f),
                                    topLeft = Offset(0f, 0f),
                                    size = Size(barW, size.height)
                                )

                                val detourX = 36.dp.toPx()
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(detourX - (barW / 2f), 0f),
                                    size = Size(barW, size.height)
                                )

                                val crossCenter = Offset(barW / 2f, centerY)
                                val crossRadius = 7.dp.toPx()
                                val crossColor = Color(0xFFE53935)
                                val crossStroke = 3.5.dp.toPx()

                                drawLine(
                                    color = crossColor,
                                    start = Offset(crossCenter.x - crossRadius, crossCenter.y - crossRadius),
                                    end = Offset(crossCenter.x + crossRadius, crossCenter.y + crossRadius),
                                    strokeWidth = crossStroke,
                                    cap = StrokeCap.Round
                                )
                                drawLine(
                                    color = crossColor,
                                    start = Offset(crossCenter.x + crossRadius, crossCenter.y - crossRadius),
                                    end = Offset(crossCenter.x - crossRadius, crossCenter.y + crossRadius),
                                    strokeWidth = crossStroke,
                                    cap = StrokeCap.Round
                                )
                            } else {
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(0f, 0f),
                                    size = Size(barW, size.height)
                                )
                                val radius = barW / 2f
                                val holeRadius = radius * 0.52f
                                drawTimelineStopDot(
                                    state = stopMarkerState,
                                    center = Offset(radius, centerY),
                                    baseRadius = holeRadius,
                                    blinkAlpha = blinkAlpha
                                )
                            }
                        }

                        val stopTextColor = when {
                            isBlocked -> tColor.copy(alpha = 0.65f)
                            else -> tColor
                        }
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(start = 12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = stop.name,
                                fontSize = 16.sp,
                                fontWeight = when {
                                    isStopHighlighted -> FontWeight.Bold
                                    isBlocked -> FontWeight.SemiBold
                                    else -> FontWeight.Medium
                                },
                                color = stopTextColor,
                                modifier = Modifier.weight(1f, fill = false)
                            )
                            val timeStr = formatTimeString(stop.arrivalTime.ifBlank { stop.departureTime })
                            val schedStopStr =
                                formatTimeString(stop.scheduledArrivalTime ?: stop.scheduledDepartureTime)
                            val isStopDelayed = stop.isDelayed || stop.delaySec >= 60
                            val isStopEarly = !isStopDelayed && stop.delaySec <= -60
                            if (timeStr.isNotBlank() && !isBlocked) {
                                Row(
                                    horizontalArrangement = Arrangement.spacedBy(5.dp),
                                    verticalAlignment = Alignment.CenterVertically,
                                    modifier = Modifier.padding(start = 8.dp)
                                ) {
                                    if ((isStopDelayed || isStopEarly) && schedStopStr.isNotBlank() && schedStopStr != timeStr) {
                                        Text(
                                            text = schedStopStr,
                                            fontSize = 12.sp,
                                            style = MaterialTheme.typography.bodySmall.copy(textDecoration = TextDecoration.LineThrough),
                                            color = Color(0xFF999999)
                                        )
                                    }
                                    val rawStopTime = stop.arrivalTime.ifBlank { stop.departureTime }
                                    val displayStopTime = if (shouldShowRemainingTime(idx + 1)) {
                                        formatRemainingStopTime(
                                            time = rawStopTime,
                                            nowMillis = nowMillis,
                                            isAtThisStop = stopMarkerState == TransitStopMarkerState.AT_STOP
                                        ) ?: timeStr
                                    } else {
                                        timeStr
                                    }
                                    Text(
                                        text = displayStopTime,
                                        fontSize = 14.sp,
                                        fontWeight = FontWeight.SemiBold,
                                        color = when {
                                            isStopDelayed -> Color(0xFFD84315)
                                            isStopEarly -> Color(0xFF2E7D32)
                                            else -> Color(0xFF666666)
                                        }
                                    )
                                }
                            }
                        }
                    }

                    val isLastBlocked =
                        isBlocked && (idx == intermediateStops.lastIndex || !intermediateStops[idx + 1].isBlocked)
                    if (isLastBlocked) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(28.dp)
                        ) {
                            Canvas(
                                modifier = Modifier
                                    .width(44.dp)
                                    .fillMaxHeight()
                            ) {
                                val barW = 16.dp.toPx()

                                drawRect(
                                    color = tColor.copy(alpha = 0.25f),
                                    topLeft = Offset(0f, 0f),
                                    size = Size(barW, size.height)
                                )
                                val detourX = 36.dp.toPx()
                                val path = Path().apply {
                                    moveTo(detourX, 0f)
                                    cubicTo(
                                        detourX, size.height * 0.5f,
                                        barW / 2f, size.height * 0.5f,
                                        barW / 2f, size.height
                                    )
                                }
                                drawPath(
                                    path = path,
                                    color = tColor,
                                    style = Stroke(
                                        width = barW,
                                        cap = StrokeCap.Round
                                    )
                                )
                            }
                        }
                    }
                }
            }
        }

        Box(
            modifier = Modifier
                .width(16.dp)
                .height(20.dp)
                .background(tColor)
        )

        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .height(IntrinsicSize.Min)
        ) {
            Canvas(
                modifier = Modifier
                    .width(16.dp)
                    .fillMaxHeight()
            ) {
                val pillWidth = size.width
                val radius = pillWidth / 2f
                val holeRadius = radius * 0.52f
                val centerY = size.height / 2f

                val topY = 0f
                val bottomY = (centerY + radius).coerceAtMost(size.height)

                val path = Path().apply {
                    addRoundRect(
                        RoundRect(
                            left = 0f,
                            top = topY,
                            right = pillWidth,
                            bottom = bottomY,
                            topLeftCornerRadius = CornerRadius.Zero,
                            topRightCornerRadius = CornerRadius.Zero,
                            bottomRightCornerRadius = CornerRadius(radius, radius),
                            bottomLeftCornerRadius = CornerRadius(radius, radius)
                        )
                    )
                }
                drawPath(path, color = tColor)

                drawTimelineStopDot(
                    state = markerStateFor(intermediateStops.size + 1),
                    center = Offset(x = radius, y = centerY),
                    baseRadius = holeRadius,
                    blinkAlpha = blinkAlpha
                )
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 12.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(
                    modifier = Modifier.weight(1f, fill = false),
                    verticalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = leg.toStopName,
                        fontSize = 20.sp,
                        fontWeight = if (isAlightingSoon) FontWeight.Bold else FontWeight.Medium,
                        color = tColor
                    )
                    val arrRef = arrivalPlatformInfo?.ref?.takeIf { it.isNotBlank() }
                        ?: leg.arrivalPlatform?.takeIf { it.isNotBlank() }
                    val exitLabel = when (arrivalPlatformInfo?.exit_side) {
                        "left" -> "Sortie à gauche"
                        "right" -> "Sortie à droite"
                        else -> null
                    }
                    if (arrRef != null || exitLabel != null) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            if (arrRef != null) {
                                Surface(
                                    shape = RoundedCornerShape(6.dp),
                                    color = tColor.copy(alpha = 0.15f)
                                ) {
                                    Text(
                                        text = "Quai $arrRef",
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Bold,
                                        color = tColor,
                                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                                    )
                                }
                            }
                            if (exitLabel != null) {
                                Surface(
                                    shape = RoundedCornerShape(6.dp),
                                    color = MaterialTheme.colorScheme.surfaceVariant
                                ) {
                                    Text(
                                        text = exitLabel,
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Medium,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                                    )
                                }
                            }
                        }
                    }
                }
                val timeStr = formatTimeString(leg.arrivalTime)
                val schedArrStr = formatTimeString(leg.scheduledArrivalTime)
                val isArrDelayed =
                    leg.arrivalDelaySec >= 60 || (leg.isDelayed && schedArrStr.isNotBlank() && schedArrStr != timeStr)
                val isArrEarly = !isArrDelayed && leg.arrivalDelaySec <= -60
                if (timeStr.isNotBlank()) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(start = 8.dp)
                    ) {
                        if ((isArrDelayed || isArrEarly) && schedArrStr.isNotBlank() && schedArrStr != timeStr) {
                            Text(
                                text = schedArrStr,
                                fontSize = 13.sp,
                                style = MaterialTheme.typography.bodySmall.copy(textDecoration = TextDecoration.LineThrough),
                                color = Color(0xFF888888)
                            )
                        }
                        val arrivalPos = intermediateStops.size + 1
                        val displayArrivalTime = if (shouldShowRemainingTime(arrivalPos)) {
                            formatRemainingStopTime(
                                time = leg.arrivalTime,
                                nowMillis = nowMillis,
                                isAtThisStop = markerStateFor(arrivalPos) == TransitStopMarkerState.AT_STOP
                            ) ?: timeStr
                        } else {
                            timeStr
                        }
                        Text(
                            text = displayArrivalTime,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            color = when {
                                isArrDelayed -> Color(0xFFD84315)
                                isArrEarly -> Color(0xFF2E7D32)
                                else -> Color(0xFF555555)
                            }
                        )
                    }
                }
            }
        }
    }
}
