package com.octara.maps.ui.screens.planner

import android.app.DatePickerDialog
import android.app.TimePickerDialog
import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.wrapContentHeight
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccessTime
import androidx.compose.material.icons.automirrored.filled.Accessible
import androidx.compose.material.icons.automirrored.filled.DirectionsBike
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.SwapVert
import android.content.Context
import androidx.compose.material.icons.automirrored.filled.DirectionsWalk
import androidx.compose.material.icons.filled.FlashOn
import androidx.compose.material.icons.filled.NearMe
import androidx.compose.material.icons.filled.Weekend
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SheetState
import androidx.compose.material3.SheetValue
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
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.input.nestedscroll.NestedScrollConnection
import androidx.compose.ui.input.nestedscroll.NestedScrollSource
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.types.JourneyQuery
import com.octara.maps.types.JourneyResponse
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.TripLeg
import com.octara.maps.types.Position
import com.octara.maps.types.TimeType
import com.octara.maps.types.Route
import com.octara.maps.types.Accessibility
import com.octara.maps.utils.LocationHelper
import com.octara.maps.utils.api
import com.octara.maps.ui.components.RouteIcon
import java.text.SimpleDateFormat
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import java.util.Calendar
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun JourneyPlannerSheetContent(
    toStopId: String,
    toStopName: String,
    modifier: Modifier = Modifier,
    sheetState: SheetState? = null,
    onClose: (() -> Unit)? = null
) {
    val router = LocalRouter.current
    val configuration = LocalConfiguration.current
    val context = LocalContext.current

    var currentCoords by remember { mutableStateOf<Pair<Double, Double>?>(null) }

    var journeyConfig by remember {
        mutableStateOf(
            JourneyQuery(
                from = Position.CurrentLocation,
                to = Position.ByStop(toStopId),
                time = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss")),
                timeType = TimeType.DEPARTURE,
                allowedModes = null,
                withBike = false,
                wheelchair = false,
                currentGps = null
            )
        )
    }

    LaunchedEffect(Unit) {
        val loc = LocationHelper(context).getCurrentLocation()
        if (loc != null) {
            val coords = Pair(loc.first.toDouble(), loc.second.toDouble())
            currentCoords = coords
            journeyConfig = journeyConfig.copy(currentGps = coords)
        } else {
            val fallback = Pair(48.8566, 2.3522)
            currentCoords = fallback
            journeyConfig = journeyConfig.copy(currentGps = fallback)
        }
    }

    var journeys by remember { mutableStateOf(JourneyResponse()) }
    var isLoading by remember { mutableStateOf(false) }

    LaunchedEffect(journeyConfig) {
        if (journeyConfig.currentGps == null) {
            return@LaunchedEffect
        }
        isLoading = true
        journeys = JourneyResponse()
        try {
            journeys = api.getJourney(query = journeyConfig)
        } catch (e: Exception) {
            e.printStackTrace()
        } finally {
            isLoading = false
        }
    }

    var selectedCalendar by remember { mutableStateOf(Calendar.getInstance()) }
    var isMenuExpanded by remember { mutableStateOf(false) }

    fun formatDisplayDateTime(cal: Calendar): String {
        val now = Calendar.getInstance()
        val isToday = now.get(Calendar.YEAR) == cal.get(Calendar.YEAR) &&
                now.get(Calendar.DAY_OF_YEAR) == cal.get(Calendar.DAY_OF_YEAR)
        val isTomorrow = now.get(Calendar.YEAR) == cal.get(Calendar.YEAR) &&
                now.get(Calendar.DAY_OF_YEAR) + 1 == cal.get(Calendar.DAY_OF_YEAR)
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        val minute = String.format(Locale.getDefault(), "%02d", cal.get(Calendar.MINUTE))
        return when {
            isToday -> "${hour}h$minute"
            isTomorrow -> "Demain ${hour}h$minute"
            else -> "${cal.get(Calendar.DAY_OF_MONTH)}/${cal.get(Calendar.MONTH) + 1} ${hour}h$minute"
        }
    }

    val openDateTimePicker: () -> Unit = {
        val datePicker = DatePickerDialog(
            context,
            { _, year, month, dayOfMonth ->
                val timePicker = TimePickerDialog(
                    context,
                    { _, hourOfDay, minute ->
                        val chosen = Calendar.getInstance().apply {
                            set(Calendar.YEAR, year)
                            set(Calendar.MONTH, month)
                            set(Calendar.DAY_OF_MONTH, dayOfMonth)
                            set(Calendar.HOUR_OF_DAY, hourOfDay)
                            set(Calendar.MINUTE, minute)
                            set(Calendar.SECOND, 0)
                        }
                        val now = Calendar.getInstance()
                        val finalCal = if (chosen.before(now)) now else chosen
                        selectedCalendar = finalCal
                        val isoFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.getDefault())
                        journeyConfig = journeyConfig.copy(time = isoFormat.format(finalCal.time))
                    },
                    selectedCalendar.get(Calendar.HOUR_OF_DAY),
                    selectedCalendar.get(Calendar.MINUTE),
                    true
                )
                timePicker.show()
            },
            selectedCalendar.get(Calendar.YEAR),
            selectedCalendar.get(Calendar.MONTH),
            selectedCalendar.get(Calendar.DAY_OF_MONTH)
        )
        datePicker.datePicker.minDate = System.currentTimeMillis() - 1000L
        datePicker.show()
    }

    var isFullscreen by remember { mutableStateOf(false) }

    BackHandler(enabled = isFullscreen) {
        isFullscreen = false
    }

    if (sheetState != null) {
        LaunchedEffect(sheetState.currentValue) {
            if (sheetState.currentValue == SheetValue.Hidden) {
                onClose?.invoke() ?: router.goBack()
            } else if (sheetState.currentValue == SheetValue.PartiallyExpanded) {
                isFullscreen = false
            }
        }
    }

    LaunchedEffect(Unit) {
        if (sheetState != null && sheetState.currentValue != SheetValue.Expanded) {
            try {
                sheetState.expand()
            } catch (_: Exception) {
            }
        }
    }

    val nestedScrollConnection = remember {
        object : NestedScrollConnection {
            override fun onPreScroll(available: Offset, source: NestedScrollSource): Offset {
                if (available.y < -15f && !isFullscreen && sheetState?.currentValue == SheetValue.Expanded) {
                    isFullscreen = true
                } else if (available.y > 15f && isFullscreen) {
                    isFullscreen = false
                }
                return Offset.Zero
            }
        }
    }

    val screenHeight = configuration.screenHeightDp.dp
    val targetHeight = if (isFullscreen) screenHeight else (screenHeight * 0.82f)
    val animatedHeight by animateDpAsState(
        targetValue = targetHeight,
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioNoBouncy,
            stiffness = Spring.StiffnessMediumLow
        ),
        label = "journeyPlannerHeight"
    )

    Column(
        modifier = modifier
            .fillMaxWidth()
            .height(animatedHeight)
            .then(if (isFullscreen) Modifier.statusBarsPadding() else Modifier)
            .navigationBarsPadding()
            .nestedScroll(nestedScrollConnection)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        val fromIsCurrentLocation = journeyConfig.from is Position.CurrentLocation
        val fromText = if (fromIsCurrentLocation) "Ma position" else toStopName
        val fromColor = if (fromIsCurrentLocation) Color(0xFF007AFF) else Color(0xFF34C759)

        val toIsCurrentLocation = journeyConfig.to is Position.CurrentLocation
        val toText = if (toIsCurrentLocation) "Ma position" else toStopName
        val toColor = if (toIsCurrentLocation) Color(0xFF007AFF) else Color(0xFF34C759)

        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(12.dp),
            color = Color(0xFFE8E8ED)
        ) {
            Box(
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                contentAlignment = Alignment.CenterEnd
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(vertical = 6.dp)
                    ) {
                        Text(
                            text = fromText,
                            color = fromColor,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold
                        )
                    }
                    HorizontalDivider(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(end = 40.dp),
                        thickness = 1.dp,
                        color = Color(0xFFE0E0E0)
                    )
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(vertical = 6.dp)
                    ) {
                        Text(
                            text = toText,
                            color = toColor,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold
                        )
                    }
                }

                Surface(
                    shape = RoundedCornerShape(percent = 50),
                    color = Color.White,
                    shadowElevation = 2.dp,
                    modifier = Modifier
                        .width(48.dp)
                        .height(32.dp)
                        .clip(RoundedCornerShape(percent = 50))
                        .clickable {
                            journeyConfig = journeyConfig.copy(
                                from = journeyConfig.to,
                                to = journeyConfig.from
                            )
                        }
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(
                            imageVector = Icons.Default.SwapVert,
                            contentDescription = "Inverser",
                            tint = Color(0xFF555555),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Surface(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { isMenuExpanded = true },
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFFE8E8ED)
            ) {
                Box(
                    modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp)
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = if (journeyConfig.timeType == TimeType.DEPARTURE) "Départ à" else "Arrivée à",
                            color = Color(0xFF555555),
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold
                        )
                        Icon(
                            imageVector = Icons.Default.ArrowDropDown,
                            contentDescription = null,
                            tint = Color(0xFF555555),
                            modifier = Modifier.size(20.dp)
                        )
                    }

                    DropdownMenu(
                        expanded = isMenuExpanded,
                        onDismissRequest = { isMenuExpanded = false }
                    ) {
                        DropdownMenuItem(
                            text = { Text("Départ à", fontWeight = FontWeight.SemiBold) },
                            onClick = {
                                journeyConfig = journeyConfig.copy(timeType = TimeType.DEPARTURE)
                                isMenuExpanded = false
                            }
                        )
                        DropdownMenuItem(
                            text = { Text("Arrivée à", fontWeight = FontWeight.SemiBold) },
                            onClick = {
                                journeyConfig = journeyConfig.copy(timeType = TimeType.ARRIVAL)
                                isMenuExpanded = false
                            }
                        )
                    }
                }
            }

            Surface(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { openDateTimePicker() },
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFFE8E8ED)
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 14.dp, vertical = 10.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = formatDisplayDateTime(selectedCalendar),
                        color = Color(0xFF555555),
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold
                    )
                    Icon(
                        imageVector = Icons.Default.AccessTime,
                        contentDescription = "Choisir l'heure",
                        tint = Color(0xFF555555),
                        modifier = Modifier.size(18.dp)
                    )
                }
            }
        }

        val prefs = remember { context.getSharedPreferences("journey_planner_prefs", Context.MODE_PRIVATE) }
        var selectedMode by remember {
            mutableStateOf(prefs.getString("preferred_route_mode", "comfort") ?: "comfort")
        }

        data class RouteModeOption(
            val key: String,
            val label: String,
            val icon: androidx.compose.ui.graphics.vector.ImageVector
        )

        val routeModes = remember {
            listOf(
                RouteModeOption("comfort", "Confort", Icons.Default.Weekend),
                RouteModeOption("fastest", "Rapide", Icons.Default.FlashOn),
                RouteModeOption("least_walk", "Moins de marche", Icons.AutoMirrored.Filled.DirectionsWalk),
                RouteModeOption("nearest_station", "Proche", Icons.Default.NearMe)
            )
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            for (mode in routeModes) {
                val isSelected = (mode.key == selectedMode)
                val trip = journeys.journeys[mode.key]
                val timeLabel = when {
                    trip == null -> ""
                    trip.durationSec < 3600 -> "${trip.durationSec / 60}m"
                    else -> "${trip.durationSec / 3600}h${(trip.durationSec % 3600) / 60}"
                }

                Surface(
                    modifier = Modifier
                        .weight(if (isSelected) 1.6f else 1f)
                        .clip(RoundedCornerShape(12.dp))
                        .clickable {
                            selectedMode = mode.key
                            prefs.edit().putString("preferred_route_mode", mode.key).apply()
                        },
                    shape = RoundedCornerShape(12.dp),
                    color = if (isSelected) MaterialTheme.colorScheme.primaryContainer else Color(0xFFE8E8ED)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 6.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.Center,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        if (isSelected) {
                            Text(
                                text = mode.label,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                        } else {
                            Icon(
                                imageVector = mode.icon,
                                contentDescription = mode.label,
                                modifier = Modifier.size(16.dp),
                                tint = Color(0xFF555555)
                            )
                            if (timeLabel.isNotEmpty()) {
                                Spacer(modifier = Modifier.width(3.dp))
                                Text(
                                    text = timeLabel,
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = Color(0xFF555555),
                                    maxLines = 1
                                )
                            }
                        }
                    }
                }
            }
        }

        HorizontalDivider(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            thickness = 1.dp,
            color = Color(0xFFE0E0E0)
        )

        val journey = journeys.journeys[selectedMode] ?: journeys.journeys["comfort"]

        LaunchedEffect(journey) {
            router.activeJourney = journey
        }

        val duration = when {
            isLoading -> "Recherche en cours..."
            journey == null -> if (journeys.journeys.isEmpty()) "Aucun itinéraire trouvé" else "Non disponible"
            journey.durationSec < 3600 -> "${journey.durationSec / 60}min"
            else -> "${journey.durationSec / 3600}h ${(journey.durationSec % 3600) / 60}min"
        }

        Text(
            text = duration,
            fontSize = 16.sp,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(vertical = 8.dp)
        )

        for (leg in journey?.legs.orEmpty()) {
            JourneyLegItem(leg = leg)
            Spacer(modifier = Modifier.height(8.dp))
        }
    }
}

@Composable
fun JourneyLegItem(leg: TripLeg) {
    if (leg.type == com.octara.maps.types.LegType.TRANSIT) {
        TransitLegItem(leg = leg)
    } else if (leg.type == com.octara.maps.types.LegType.BIKE) {
        WalkingLegItem(leg = leg)
    } else {
        WalkingLegItem(leg = leg)
    }
}

fun distanceToString(distanceKm: Double): String {
    return if (distanceKm < 1) {
        "${(distanceKm * 1000).toInt()} m"
    } else {
        String.format("%.1f km", distanceKm)
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
fun WalkingLegItem(leg: TripLeg) {
    val tColor = Color(0xFF757575)
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

        Text(
            text = "Marcher durant ${distanceToString(leg.distanceKm)}",
            fontSize = 20.sp,
            fontWeight = FontWeight.Medium,
            color = tColor,
            modifier = Modifier.padding(start = 12.dp)
        )
    }
}

@Composable
fun TransitLegItem(leg: TripLeg) {
    val route = leg.route
    val rgb = route?.color?.removePrefix("#")?.toLongOrNull(16)?.toInt()
        ?: 0x0071BD
    val tColor = Color(rgb or 0xFF000000.toInt())

    data class StopInfo(
        val name: String,
        val arrivalTime: String? = null,
        val headsign: String? = null,
        val wheelchair: Accessibility = Accessibility.UNKNOWN,
        val bikesAllowed: Accessibility = Accessibility.UNKNOWN
    )

    var stops: List<StopInfo> = listOf(
        StopInfo(
            name = leg.fromStopName,
            headsign = route?.longName,
            arrivalTime = leg.departureTime,
            wheelchair = leg.wheelchair,
            bikesAllowed = leg.bikesAllowed
        ),
        StopInfo(name = leg.toStopName, arrivalTime = leg.arrivalTime)
    )

    val intermediateStops = leg.intermediateStops ?: emptyList()
    var isIntermediateExpanded by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
    ) {
        val barWidth = 16.dp
        val iconHeight = 24.dp
        val iconLeftOffset = (barWidth - iconHeight) / 2

        if (route != null) {
            Box(
                modifier = Modifier
                    .offset(x = iconLeftOffset)
                    .padding(bottom = 6.dp)
            ) {
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
                drawPath(path, color = tColor)

                drawCircle(
                    color = Color.White,
                    radius = holeRadius,
                    center = Offset(x = radius, y = centerY)
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
                    color = tColor,
                    modifier = Modifier.weight(1f, fill = false)
                )
                val timeStr = formatTimeString(leg.departureTime)
                if (timeStr.isNotBlank()) {
                    Text(
                        text = timeStr,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFF555555),
                        modifier = Modifier.padding(start = 8.dp)
                    )
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
                        .background(tColor)
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
                        text = route.longName,
                        fontSize = 16.sp,
                        fontWeight = FontWeight.Normal,
                        color = tColor
                    )
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
                        .background(tColor)
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

        if (intermediateStops.isNotEmpty()) {
            Box(
                modifier = Modifier
                    .width(16.dp)
                    .height(14.dp)
                    .background(tColor)
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
                        color = tColor,
                        topLeft = Offset(0f, 0f),
                        size = androidx.compose.ui.geometry.Size(pillWidth, size.height)
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
                val hasBlocked = intermediateStops.any { it.isBlocked }

                for (idx in intermediateStops.indices) {
                    val stop = intermediateStops[idx]
                    val isBlocked = stop.isBlocked

                    val isFirstBlocked = isBlocked && (idx == 0 || !intermediateStops[idx - 1].isBlocked)

                    if (isFirstBlocked) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(IntrinsicSize.Min)
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
                                        size = androidx.compose.ui.geometry.Size(barW, size.height)
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
                                        style = androidx.compose.ui.graphics.drawscope.Stroke(
                                            width = barW,
                                            cap = androidx.compose.ui.graphics.StrokeCap.Round
                                        )
                                    )
                                }
                            }

                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color.Black,
                                modifier = Modifier.padding(start = 8.dp, top = 2.dp, bottom = 2.dp)
                            ) {
                                Text(
                                    text = "Situation anormale",
                                    color = Color.White,
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
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
                                .width(if (hasBlocked) 44.dp else 16.dp)
                                .fillMaxHeight()
                        ) {
                            val barW = 16.dp.toPx()
                            if (hasBlocked) {
                                drawRect(
                                    color = tColor.copy(alpha = 0.25f),
                                    topLeft = Offset(0f, 0f),
                                    size = androidx.compose.ui.geometry.Size(barW, size.height)
                                )

                                if (isBlocked) {
                                    val detourX = 36.dp.toPx()
                                    drawRect(
                                        color = tColor,
                                        topLeft = Offset(detourX - (barW / 2f), 0f),
                                        size = androidx.compose.ui.geometry.Size(barW, size.height)
                                    )
                                }
                            } else {
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(0f, 0f),
                                    size = androidx.compose.ui.geometry.Size(barW, size.height)
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
                                .width(if (hasBlocked) 44.dp else 16.dp)
                                .fillMaxHeight()
                        ) {
                            val barW = 16.dp.toPx()
                            val centerY = size.height / 2f

                            if (hasBlocked) {
                                drawRect(
                                    color = tColor.copy(alpha = 0.25f),
                                    topLeft = Offset(0f, 0f),
                                    size = androidx.compose.ui.geometry.Size(barW, size.height)
                                )

                                if (isBlocked) {
                                    val detourX = 36.dp.toPx()
                                    drawRect(
                                        color = tColor,
                                        topLeft = Offset(detourX - (barW / 2f), 0f),
                                        size = androidx.compose.ui.geometry.Size(barW, size.height)
                                    )

                                    val crossCenter = Offset(barW / 2f, centerY)
                                    val crossRadius = 6.dp.toPx()
                                    val crossColor = Color(0xFFE53935)
                                    val crossStroke = 2.5.dp.toPx()

                                    drawLine(
                                        color = crossColor,
                                        start = Offset(crossCenter.x - crossRadius, crossCenter.y - crossRadius),
                                        end = Offset(crossCenter.x + crossRadius, crossCenter.y + crossRadius),
                                        strokeWidth = crossStroke,
                                        cap = androidx.compose.ui.graphics.StrokeCap.Round
                                    )
                                    drawLine(
                                        color = crossColor,
                                        start = Offset(crossCenter.x + crossRadius, crossCenter.y - crossRadius),
                                        end = Offset(crossCenter.x - crossRadius, crossCenter.y + crossRadius),
                                        strokeWidth = crossStroke,
                                        cap = androidx.compose.ui.graphics.StrokeCap.Round
                                    )
                                } else {
                                    val radius = barW / 2f
                                    val holeRadius = radius * 0.52f
                                    drawCircle(
                                        color = Color.White,
                                        radius = holeRadius,
                                        center = Offset(radius, centerY)
                                    )
                                }
                            } else {
                                drawRect(
                                    color = tColor,
                                    topLeft = Offset(0f, 0f),
                                    size = androidx.compose.ui.geometry.Size(barW, size.height)
                                )
                                val radius = barW / 2f
                                val holeRadius = radius * 0.52f
                                drawCircle(
                                    color = Color.White,
                                    radius = holeRadius,
                                    center = Offset(radius, centerY)
                                )
                            }
                        }

                        val stopTextColor = if (isBlocked) tColor.copy(alpha = 0.35f) else tColor
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
                                fontWeight = if (isBlocked) FontWeight.Normal else FontWeight.Medium,
                                color = stopTextColor,
                                modifier = Modifier.weight(1f, fill = false)
                            )
                            val timeStr = formatTimeString(stop.arrivalTime.ifBlank { stop.departureTime })
                            if (timeStr.isNotBlank() && !isBlocked) {
                                Text(
                                    text = timeStr,
                                    fontSize = 14.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = Color(0xFF666666),
                                    modifier = Modifier.padding(start = 8.dp)
                                )
                            }
                        }
                    }

                    val isLastBlocked =
                        isBlocked && (idx == intermediateStops.lastIndex || !intermediateStops[idx + 1].isBlocked)
                    if (isLastBlocked) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(22.dp)
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
                                    size = androidx.compose.ui.geometry.Size(barW, size.height)
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
                                    style = androidx.compose.ui.graphics.drawscope.Stroke(
                                        width = barW,
                                        cap = androidx.compose.ui.graphics.StrokeCap.Round
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

                drawCircle(
                    color = Color.White,
                    radius = holeRadius,
                    center = Offset(x = radius, y = centerY)
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
                    text = leg.toStopName,
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Medium,
                    color = tColor,
                    modifier = Modifier.weight(1f, fill = false)
                )
                val timeStr = formatTimeString(leg.arrivalTime)
                if (timeStr.isNotBlank()) {
                    Text(
                        text = timeStr,
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFF555555),
                        modifier = Modifier.padding(start = 8.dp)
                    )
                }
            }
        }
    }
}